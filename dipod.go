package dipod

import (
	"context"
	"errors"

	"github.com/EricHripko/dipod/iopodman"
)

// DockerUnixAddress is the default address of the Docker unix socket.
const DockerUnixAddress = "/var/run/docker.sock"

// PodmanUnixAddress is the default address of the Podman unix socket.
const PodmanUnixAddress = "unix:///run/podman/io.podman"

// APIVersion defines maximum Docker Engine API version supported by this
// proxy.
const APIVersion = "1.40"

// MinAPIVersion defines minimum Docker Engine API version supported by this
// proxy.
const MinAPIVersion = "1.12"

// ProxyVersion identified the dipod version.
const ProxyVersion = "0.0.1"

// ErrNotImplemented is returned when functionality requested was not
// implemented yet.
var ErrNotImplemented = errors.New("dipod: not implemented")

type recvFunc func(ctx context.Context) (iopodman.MoreResponse, uint64, error)
