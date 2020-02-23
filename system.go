package dipod

import (
	"context"
	"net"
	"net/http"
	"runtime"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/registry"
	log "github.com/sirupsen/logrus"

	"github.com/EricHripko/dipod/iopodman"
)

const id = "5H6A:ME4Z:MBS5:AEUT:BDYB:MBHM:Y6UI:Y7CZ:DOGT:2CXX:D5RG:BKCP"

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

// SystemInfo is a handler function for /info.
func SystemInfo(res http.ResponseWriter, req *http.Request) {
	log.Debug("system info")
	backend, err := iopodman.GetInfo().Call(context.TODO(), podman)
	if err != nil {
		WriteError(res, http.StatusInternalServerError, err)
		return
	}

	info := types.Info{
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
	JSONResponse(res, info)
}
