package dipod

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"io/ioutil"
	"os"

	"github.com/EricHripko/dipod/iopodman"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/backend"
	"github.com/varlink/go/varlink"
)

type buildsBackend struct{}

func (*buildsBackend) Build(ctx context.Context, config backend.BuildConfig) (id string, err error) {
	if len(config.Options.Tags) < 1 {
		err = errors.New("dipod: cannot build image without tag")
		return
	}

	// stash context tarball to a temp file
	var buildContext *os.File
	buildContext, err = ioutil.TempFile("", "dipod-build")
	if err != nil {
		return
	}
	io.Copy(buildContext, config.Source)
	defer config.Source.Close()
	defer os.Remove(buildContext.Name())

	// translate options
	opts := iopodman.BuildInfo{
		AdditionalTags: config.Options.Tags[1:],
		BuildArgs:      make(map[string]string),
		BuildOptions: iopodman.BuildOptions{
			AddHosts:     config.Options.ExtraHosts,
			CgroupParent: config.Options.CgroupParent,
			CpuPeriod:    config.Options.CPUPeriod,
			CpuQuota:     config.Options.CPUQuota,
			CpuShares:    config.Options.CPUShares,
			CpusetCpus:   config.Options.CPUSetCPUs,
			CpusetMems:   config.Options.CPUSetMems,
			Memory:       config.Options.Memory,
			MemorySwap:   config.Options.MemorySwap,
		},
		ContextDir:              buildContext.Name(),
		Dockerfiles:             []string{config.Options.Dockerfile},
		ForceRmIntermediateCtrs: config.Options.ForceRemove,
		Nocache:                 config.Options.NoCache,
		Squash:                  config.Options.Squash,
		Output:                  config.Options.Tags[0],
	}
	for name, value := range config.Options.BuildArgs {
		opts.BuildArgs[name] = *value
	}
	for _, ulimit := range config.Options.Ulimits {
		opts.BuildOptions.Ulimit = append(opts.BuildOptions.Ulimit, ulimit.String())
	}

	for name, value := range config.Options.Labels {
		opts.Label = append(opts.Label, name+"="+value)
	}
	if config.Options.PullParent {
		opts.PullPolicy = "PullAlways"
	}

	json.NewEncoder(os.Stdout).Encode(config.Options)

	// build
	var recv recvFunc
	recv, err = iopodman.BuildImage().Send(ctx, podman, varlink.More, opts)
	if err != nil {
		return
	}
	for {
		var status iopodman.MoreResponse
		var flags uint64
		status, flags, err = recv(ctx)
		if err != nil {
			return
		}

		id = status.Id
		for _, log := range status.Logs {
			_, err = config.ProgressWriter.StdoutFormatter.Write([]byte(log))
			if err != nil {
				return
			}
		}

		if flags&varlink.Continues != varlink.Continues {
			break
		}
	}
	return
}

func (*buildsBackend) PruneCache(context.Context, types.BuildCachePruneOptions) (*types.BuildCachePruneReport, error) {
	return nil, errors.New("not implemented")
}

func (*buildsBackend) Cancel(context.Context, string) error {
	return errors.New("not implemented")
}
