module github.com/EricHripko/dipod

go 1.12

require (
	github.com/coreos/go-systemd v0.0.0-20190719114852-fd7a80b32e1f
	github.com/docker/distribution v2.7.1+incompatible
	github.com/docker/docker v1.13.1
	github.com/docker/go-connections v0.4.0
	github.com/docker/go-units v0.4.0 // indirect
	github.com/gorilla/mux v1.7.3
	github.com/moby/moby v1.13.1
	github.com/opencontainers/go-digest v1.0.0-rc1 // indirect
	github.com/sirupsen/logrus v1.4.2
	github.com/varlink/go v0.0.0-20190502142041-0f1d566d194b
)

replace github.com/Sirupsen/logrus v1.4.2 => github.com/sirupsen/logrus v1.4.2
