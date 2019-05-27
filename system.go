package dipod

import (
	"net/http"
	"runtime"

	"github.com/moby/moby/api/types"
	log "github.com/sirupsen/logrus"
)

// Ping is a handler function for /_ping.
func Ping(res http.ResponseWriter, req *http.Request) {
	log.WithField("api-version", APIVersion).Debug("ping")
	res.Header().Add("API-Version", APIVersion)
	res.Header().Add("Docker-Experimental", "false")
	res.WriteHeader(http.StatusOK)
}

// Version is a handler function for /version.
func Version(res http.ResponseWriter, req *http.Request) {
	log.WithField("version", Version).Debug("version")
	ver := types.Version{
		APIVersion:    APIVersion,
		Arch:          runtime.GOARCH,
		BuildTime:     "",
		Experimental:  false,
		GitCommit:     "",
		GoVersion:     runtime.Version(),
		KernelVersion: "n/a",
		MinAPIVersion: APIVersion,
		Os:            runtime.GOOS,
		Version:       ProxyVersion + "-dipod",
	}
	JSONResponse(res, ver)
}
