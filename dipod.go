package dipod

// MobyUnixAddress is the default address of the Moby unix socket.
const MobyUnixAddress = "/var/run/docker.sock"

// PodmanUnixAddress is the default address of the Podman unix socket.
const PodmanUnixAddress = "unix:///run/podman/io.podman"

// APIVersion defines Moby Engine API version supported by this proxy.
const APIVersion = "1.26"

// ProxyVersion identified the dipod version.
const ProxyVersion = "0.0.1"
