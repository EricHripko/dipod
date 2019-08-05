package dipod

import (
	"net"
	"net/http"

	"github.com/coreos/go-systemd/activation"
	log "github.com/sirupsen/logrus"
)

// Serve starts a Moby Engine proxy.
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
		listener, err = net.Listen("unix", MobyUnixAddress)
		if err != nil {
			log.WithError(err).Fatal("unix listen fail")
		}
	}
	defer listener.Close()
	log.WithField("address", listener.Addr()).Info("unix listen")

	// unhandled request
	http.HandleFunc("/", func(res http.ResponseWriter, req *http.Request) {
		log.WithField("uri", req.RequestURI).Warn("not implemented")
		res.WriteHeader(http.StatusNotImplemented)
	})
	// system
	http.HandleFunc("/_ping", Ping)
	http.HandleFunc("/v1.26/version", Version)
	http.HandleFunc("/v1.26/info", SystemInfo)
	// images
	http.HandleFunc("/v1.26/images/json", ImageList)
	http.HandleFunc("/v1.26/build", ImageBuild)
	http.HandleFunc("/v1.26/images/create", ImageCreate)

	err = http.Serve(listener, nil)
	if err != nil {
		log.WithError(err).Fatal("http server fail")
	}
}
