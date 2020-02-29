package dipod

import (
	"context"
	"errors"
	"net"
	"runtime"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/events"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/api/types/registry"
	"github.com/docker/docker/api/types/swarm"
	log "github.com/sirupsen/logrus"

	"github.com/EricHripko/dipod/iopodman"
)

const id = "5H6A:ME4Z:MBS5:AEUT:BDYB:MBHM:Y6UI:Y7CZ:DOGT:2CXX:D5RG:BKCP"

type systemBackend struct {
}

func (*systemBackend) SystemInfo() (info *types.Info, err error) {
	var backend iopodman.PodmanInfo
	backend, err = iopodman.GetInfo().Call(context.TODO(), podman)
	if err != nil {
		return
	}

	info = &types.Info{
		Architecture:      backend.Host.Arch,
		BridgeNfIP6tables: true,
		BridgeNfIptables:  true,
		CPUCfsPeriod:      true,
		CPUCfsQuota:       true,
		CPUSet:            true,
		CPUShares:         true,
		CgroupDriver:      "podman",
		Containers:        int(backend.Store.Containers),
		Debug:             false,
		DockerRootDir:     backend.Store.Run_root,
		Driver:            backend.Store.Graph_driver_name,
		ID:                id,
		IPv4Forwarding:    true,
		Images:            int(backend.Store.Images),
		KernelMemory:      true,
		KernelVersion:     backend.Host.Kernel,
		MemoryLimit:       true,
		MemTotal:          backend.Host.Mem_total,
		NCPU:              int(backend.Host.Cpus),
		Name:              backend.Host.Hostname,
		OSType:            backend.Host.Os,
		OomKillDisable:    true,
		OperatingSystem: (backend.Host.Distribution.Distribution + " " +
			backend.Host.Distribution.Version),
		RegistryConfig: &registry.ServiceConfig{},
		ServerVersion:  backend.Podman.Podman_version,
		SwapLimit:      true,
		SystemTime:     time.Now().Format(time.RFC3339Nano),
	}
	info.DriverStatus = append(
		info.DriverStatus,
		[...]string{"Root Dir", backend.Store.Graph_root},
		[...]string{"Options", backend.Store.Graph_driver_options},
		[...]string{"Backing Filesystem", backend.Store.Graph_status.Backing_filesystem},
		[...]string{"Supports d_type", backend.Store.Graph_status.Supports_d_type},
		[...]string{"Native Overlay Diff", backend.Store.Graph_status.Native_overlay_diff},
	)
	for _, cidr := range backend.Insecure_registries {
		_, ipnet, err := net.ParseCIDR(cidr)
		if err != nil {
			log.WithField("cidr", cidr).Warn("cidr parse fail")
			continue
		}

		netipnet := registry.NetIPNet(*ipnet)
		info.RegistryConfig.InsecureRegistryCIDRs = append(
			info.RegistryConfig.InsecureRegistryCIDRs,
			&netipnet,
		)
	}
	return
}

func (*systemBackend) SystemVersion() types.Version {
	return types.Version{
		APIVersion:    APIVersion,
		Arch:          runtime.GOARCH,
		BuildTime:     "",
		Experimental:  false,
		GitCommit:     "",
		GoVersion:     runtime.Version(),
		KernelVersion: "n/a",
		MinAPIVersion: MinAPIVersion,
		Os:            runtime.GOOS,
		Version:       ProxyVersion + "-dipod",
	}
}

func (*systemBackend) SystemDiskUsage(ctx context.Context) (*types.DiskUsage, error) {
	return nil, errors.New("not implemented")
}

func (*systemBackend) SubscribeToEvents(since, until time.Time, ef filters.Args) ([]events.Message, chan interface{}) {
	return nil, nil
}

func (*systemBackend) UnsubscribeFromEvents(chan interface{}) {

}

func (*systemBackend) AuthenticateToRegistry(ctx context.Context, authConfig *types.AuthConfig) (string, string, error) {
	return "", "", nil
}

func (*systemBackend) Info() swarm.Info {
	return swarm.Info{}
}
