package dipod

import (
	"context"

	"github.com/EricHripko/dipod/iopodman"
	log "github.com/sirupsen/logrus"
	"github.com/varlink/go/varlink"
)

var podman *varlink.Connection

// Connect to the Podman's varlink interface.
func Connect() {
	var err error
	podman, err = varlink.NewConnection(context.TODO(), PodmanUnixAddress)
	log := log.WithField("address", PodmanUnixAddress)
	if err != nil {
		log.WithError(err).Fatal("podman connect fail")
	} else {
		log.Info("podman connected")
	}
}

// ErrorMessage returns a human-readable error message from go/varlink error.
func ErrorMessage(err error) string {
	if podErr, ok := err.(*iopodman.ErrorOccurred); ok {
		return podErr.Reason
	}
	return err.Error()
}
