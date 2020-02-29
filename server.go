package dipod

import (
	"net"

	"github.com/coreos/go-systemd/activation"
	log "github.com/sirupsen/logrus"

	"github.com/docker/docker/api/server"
	"github.com/docker/docker/api/server/middleware"
	"github.com/docker/docker/api/server/router/build"
	"github.com/docker/docker/api/server/router/image"
	"github.com/docker/docker/api/server/router/system"
)

// Serve starts a Docker Engine proxy.
func Serve() {
	var (
		listeners []net.Listener
		listener  net.Listener
		err       error
	)
	listeners, err = activation.Listeners()
	if err != nil {
		log.WithError(err).Warn("systemd activation fail")
	}
	if len(listeners) > 0 {
		listener = listeners[0]
	} else {
		listener, err = net.Listen("unix", DockerUnixAddress)
		if err != nil {
			log.WithError(err).Fatal("unix listen fail")
		}
	}
	defer listener.Close()
	log.WithField("address", listener.Addr()).Info("unix listen")
	server := server.New(&server.Config{Logging: true, Version: APIVersion})
	server.Accept("", listener)

	features := make(map[string]bool)
	// build
	builds := &buildsBackend{}
	server.InitRouter(build.NewRouter(builds, nil, &features))
	// images
	images := &imageBackend{}
	server.InitRouter(image.NewRouter(images))
	// system
	sys := &systemBackend{}
	server.InitRouter(system.NewRouter(sys, sys, nil, &features))

	// middleware
	server.UseMiddleware(
		middleware.NewVersionMiddleware(ProxyVersion, APIVersion, MinAPIVersion),
	)

	wait := make(chan error)
	go server.Wait(wait)

	err = <-wait
	if err != nil {
		log.WithError(err).Fatal("docker server fail")
	}
}
