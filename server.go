package dipod

import (
	"net"
	"net/http"

	"github.com/coreos/go-systemd/activation"
	"github.com/gorilla/mux"
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

	r := mux.NewRouter()
	// unhandled request
	r.HandleFunc("/", func(res http.ResponseWriter, req *http.Request) {
		log.WithField("uri", req.RequestURI).Warn("not implemented")
		res.WriteHeader(http.StatusNotImplemented)
	})
	// system
	r.HandleFunc("/_ping", Ping)
	r.HandleFunc("/v1.26/version", Version)
	r.HandleFunc("/v1.26/info", SystemInfo)
	// images
	r.HandleFunc("/v1.26/images/json", ImageList)
	r.HandleFunc("/v1.26/build", ImageBuild)
	r.HandleFunc("/v1.26/images/create", ImageCreate)
	r.HandleFunc("/v1.26/images/{name}/json", ImageInspect)
	r.HandleFunc("/v1.26/images/{name}/history", ImageHistory)
	r.HandleFunc("/v1.26/images/{name}/tag", ImageTag)
	r.HandleFunc("/v1.26/images/{name}", ImageDelete).Methods("DELETE")
	r.HandleFunc("/v1.26/images/search", ImageSearch)
	r.HandleFunc("/v1.26/images/{name}/get", ImageGet)
	r.HandleFunc("/v1.26/images/get", ImageGetAll)

	err = http.Serve(listener, r)
	if err != nil {
		log.WithError(err).Fatal("http server fail")
	}
}
