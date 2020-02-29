package dipod

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"io/ioutil"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/EricHripko/dipod/iopodman"

	"github.com/docker/distribution/reference"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/api/types/image"
	"github.com/docker/docker/api/types/registry"
	"github.com/docker/docker/pkg/jsonmessage"
	"github.com/docker/go-connections/nat"
	specs "github.com/opencontainers/image-spec/specs-go/v1"
	log "github.com/sirupsen/logrus"
	"github.com/varlink/go/varlink"
)

type imageBackend struct {
}

func (*imageBackend) ImageDelete(imageRef string, force, prune bool) (res []types.ImageDeleteResponseItem, err error) {
	item := types.ImageDeleteResponseItem{}
	item.Deleted, err = iopodman.RemoveImage().Call(
		context.TODO(),
		podman,
		imageRef,
		force,
	)
	if err != nil {
		return
	}

	res = append(res, item)
	return
}

func (*imageBackend) ImageHistory(imageName string) (res []*image.HistoryResponseItem, err error) {
	var history []iopodman.ImageHistory
	history, err = iopodman.HistoryImage().Call(context.TODO(), podman, imageName)
	if err != nil {
		return
	}

	for _, l := range history {
		layer := &image.HistoryResponseItem{
			ID:        l.Id,
			CreatedBy: l.CreatedBy,
			Tags:      l.Tags,
			Size:      l.Size,
			Comment:   l.Comment,
		}
		if created, err := time.Parse(time.RFC3339, l.Created); err == nil {
			layer.Created = created.Unix()
		} else {
			log.
				WithError(err).
				WithField("created", l.Created).Warn("created parse fail")
		}

		res = append(res, layer)
	}
	return
}

func (*imageBackend) Images(imageFilters filters.Args, all bool, withExtraAttrs bool) (images []*types.ImageSummary, err error) {
	if all {
		err = errors.New("not implemented")
		return
	}

	var srcs []iopodman.Image
	srcs, err = iopodman.ListImages().Call(context.TODO(), podman)
	if err != nil {
		return
	}

	for _, src := range srcs {
		if imageFilters.Contains("label") && !imageFilters.MatchKVList("label", src.Labels) {
			continue
		}
		if imageFilters.Contains("reference") {
			matched := false
			for _, search := range imageFilters.Get("reference") {
				if matched {
					break
				}

				params := strings.Split(search, ":")
				var (
					id  string
					tag string
				)
				if len(params) == 0 {
					continue
				}
				id = params[0]
				if len(params) > 1 {
					tag = params[1]
				}
				for _, rt := range src.RepoTags {
					if strings.HasPrefix(rt, id+":") {
						if tag == "" {
							matched = true
						} else {
							if strings.HasSuffix(rt, ":"+tag) {
								matched = true
							}
						}
					}
				}
			}
			if !matched {
				continue
			}
		}

		image := &types.ImageSummary{
			Containers:  src.Containers,
			Created:     0,
			ID:          src.Id,
			Labels:      src.Labels,
			ParentID:    src.ParentId,
			RepoDigests: src.RepoDigests,
			RepoTags:    src.RepoTags,
			Size:        src.Size,
			SharedSize:  0,
			VirtualSize: src.VirtualSize,
		}
		if image.RepoTags == nil {
			image.RepoTags = []string{"<none>:<none>"}
		}
		if image.RepoDigests == nil {
			image.RepoDigests = []string{"<none>@<none>"}
		}
		if created, err := time.Parse(time.RFC3339, src.Created); err == nil {
			image.Created = created.Unix()
		} else {
			log.
				WithError(err).
				WithField("created", src.Created).Warn("created parse fail")
		}

		images = append(images, image)
	}
	return
}

func is2ss(i []interface{}) (s []string) {
	for _, ii := range i {
		s = append(s, ii.(string))
	}
	return
}

func (*imageBackend) LookupImage(name string) (image *types.ImageInspect, err error) {
	// podman for some reason returns this as JSON string, need to decode
	var payload string
	payload, err = iopodman.InspectImage().Call(context.TODO(), podman, name)
	if err != nil {
		return
	}

	data := make(map[string]interface{})
	err = json.Unmarshal([]byte(payload), &data)
	if err != nil {
		return
	}

	digest := strings.TrimPrefix(data["Digest"].(string), "sha256:")
	image = &types.ImageInspect{
		ID:           data["Id"].(string),
		Container:    digest,
		Comment:      data["Comment"].(string),
		Os:           data["Os"].(string),
		Architecture: data["Architecture"].(string),
		Parent:       data["Parent"].(string),
		Config: &container.Config{
			Hostname:        "",
			Domainname:      "",
			AttachStdout:    false,
			AttachStdin:     false,
			AttachStderr:    false,
			OpenStdin:       false,
			StdinOnce:       false,
			ArgsEscaped:     true,
			NetworkDisabled: false,
			OnBuild:         nil, //todo
			Image:           digest,
			User:            "",
			WorkingDir:      "",
			MacAddress:      "",
			Entrypoint:      nil,
			Labels:          nil, //todo
		},
		DockerVersion: data["Version"].(string),
		VirtualSize:   int64(data["VirtualSize"].(float64)),
		Size:          int64(data["Size"].(float64)),
		Author:        data["Author"].(string),
		Created:       data["Created"].(string),
		RepoDigests:   is2ss(data["RepoDigests"].([]interface{})),
		RepoTags:      is2ss(data["RepoTags"].([]interface{})),
	}

	// container config
	config := data["Config"].(map[string]interface{})
	if env, ok := config["Env"]; ok {
		image.Config.Env = is2ss(env.([]interface{}))
	}
	if cmd, ok := config["Cmd"]; ok {
		image.Config.Cmd = is2ss(cmd.([]interface{}))
	}
	if ep, ok := config["Entrypoint"]; ok {
		image.Config.Entrypoint = is2ss(ep.([]interface{}))
	}
	if workdir, ok := config["WorkingDir"]; ok {
		image.Config.WorkingDir = workdir.(string)
	}
	if user, ok := config["User"]; ok {
		image.Config.User = user.(string)
	}
	if stopSignal, ok := config["StopSignal"]; ok {
		image.Config.StopSignal = stopSignal.(string)
	}
	if tmp, ok := config["ExposedPorts"]; ok {
		image.Config.ExposedPorts = make(nat.PortSet)
		ports := tmp.(map[string]interface{})
		for port := range ports {
			image.Config.ExposedPorts[nat.Port(port)] = struct{}{}
		}
	}
	if tmp, ok := config["Volumes"]; ok {
		image.Config.Volumes = make(map[string]struct{})
		vols := tmp.(map[string]interface{})
		for vol := range vols {
			image.Config.Volumes[vol] = struct{}{}
		}
	}
	if tmp, ok := config["Labels"]; ok {
		image.Config.Labels = make(map[string]string)
		labels := tmp.(map[string]interface{})
		for key, val := range labels {
			image.Config.Labels[key] = val.(string)
		}
	}
	image.ContainerConfig = image.Config

	// graph driver
	gd := data["GraphDriver"].(map[string]interface{})
	gdd := gd["Data"].(map[string]interface{})
	image.GraphDriver = types.GraphDriverData{
		Name: gd["Name"].(string),
		Data: make(map[string]string),
	}
	for key, val := range gdd {
		image.GraphDriver.Data[key] = val.(string)
	}

	// rootfs
	rootfs := data["RootFS"].(map[string]interface{})
	image.RootFS = types.RootFS{
		Type:   rootfs["Type"].(string),
		Layers: is2ss(rootfs["Layers"].([]interface{})),
	}
	return
}

func (*imageBackend) TagImage(imageName, repository, tag string) (out string, err error) {
	target := repository
	if tag != "" {
		target += ":" + tag
	}
	log := log.WithField("source", imageName).WithField("target", target)
	if target == "" {
		err = errors.New("dipod: empty target")
		log.WithError(err).Error("image tag fail")
		return
	}
	log.Debug("image tag")

	out, err = iopodman.TagImage().Call(context.TODO(), podman, imageName, target)
	return
}

func (*imageBackend) ImagesPrune(ctx context.Context, pruneFilters filters.Args) (*types.ImagesPruneReport, error) {
	return nil, errors.New("not implemented")
}

func (*imageBackend) LoadImage(inTar io.ReadCloser, outStream io.Writer, quiet bool) error {
	return errors.New("not implemented")
}

func (*imageBackend) ImportImage(src string, repository, platform string, tag string, msg string, inConfig io.ReadCloser, outStream io.Writer, changes []string) error {
	return errors.New("not implemented")
}

func (*imageBackend) ExportImage(names []string, outStream io.Writer) error {
	// prepare temp file for the tarball
	tmp, err := ioutil.TempFile("", "dipod-export")
	if err != nil {
		return err
	}
	dest := "docker-archive://" + tmp.Name()
	err = tmp.Close()
	if err != nil {
		return err
	}
	defer os.Remove(tmp.Name())

	// parse list of image names into name + list of tags
	ref, err := reference.Parse(names[0])
	if err != nil {
		return err
	}
	named, ok := ref.(reference.Named)
	if !ok {
		return errors.New("dipod: main name parse fail")
	}
	var tags []string
	for _, name := range names[1:] {
		ref, err := reference.Parse(name)
		if err != nil {
			return err
		}
		nt, ok := ref.(reference.NamedTagged)
		if !ok {
			return errors.New("dipod: secondary name parse fail")
		}
		if named.Name() != nt.Name() {
			return errors.New("dipod: multiple image export not supported")
		}
		tags = append(tags, nt.Tag())
	}

	_, err = iopodman.ExportImage().Call(context.TODO(), podman, names[0], dest, false, tags)
	if err != nil {
		return err
	}
	tmp, err = os.Open(tmp.Name())
	if err != nil {
		return err
	}
	_, err = io.Copy(outStream, tmp)
	return err
}

func (*imageBackend) PullImage(ctx context.Context, image, tag string, platform *specs.Platform, metaHeaders map[string][]string, authConfig *types.AuthConfig, outStream io.Writer) error {
	name := image + ":" + tag
	// no slash => pulling from DockerHub, docker cli shadily strips docker.io/
	// prefix even if user explicitly specified it
	if !strings.ContainsAny(name, "/") {
		name = "docker.io/library/" + name
	}

	recv, err := iopodman.PullImage().Send(context.TODO(), podman, varlink.More, name)
	if err != nil {
		return err
	}

	json := json.NewEncoder(outStream)
	for {
		status, flags, err := recv(context.TODO())
		if err != nil {
			return err
		}

		for _, log := range status.Logs {
			msg := jsonmessage.JSONMessage{
				Stream: log,
				ID:     status.Id,
			}
			err = json.Encode(msg)
			if err != nil {
				return err
			}
		}

		if flags&varlink.Continues != varlink.Continues {
			break
		}
	}
	return nil
}

func (*imageBackend) PushImage(ctx context.Context, image, tag string, metaHeaders map[string][]string, authConfig *types.AuthConfig, outStream io.Writer) error {
	return errors.New("not implemented")
}

const (
	isAutomated = "is-automated"
	isOfficial  = "is-official"
	valueYes    = "true"
	valueNo     = "false"
)

func (*imageBackend) SearchRegistryForImages(ctx context.Context, filtersArgs string, term string, limit int, authConfig *types.AuthConfig, metaHeaders map[string][]string) (res *registry.SearchResults, err error) {
	var args filters.Args
	args, err = filters.FromJSON(filtersArgs)
	if err != nil {
		return
	}

	yes := true
	no := false
	filter := iopodman.ImageSearchFilter{}
	if args.Contains(isAutomated) {
		if args.ExactMatch(isAutomated, valueYes) {
			filter.Is_automated = &yes
		}
		if args.ExactMatch(isAutomated, valueNo) {
			filter.Is_automated = &no
		}
	}
	if args.Contains(isOfficial) {
		if args.ExactMatch(isOfficial, valueYes) {
			filter.Is_official = &yes
		}
		if args.ExactMatch(isOfficial, valueNo) {
			filter.Is_official = &no
		}
	}
	stars := args.Get("stars")
	if len(stars) > 0 {
		var starNo int
		starNo, err = strconv.Atoi(stars[0])
		if err != nil {
			return
		}
		filter.Star_count = int64(starNo)
	}

	var images []iopodman.ImageSearchResult
	limit64 := int64(limit)
	images, err = iopodman.SearchImages().Call(ctx, podman, term, &limit64, filter)
	if err != nil {
		return
	}

	res = &registry.SearchResults{
		Query:      term,
		NumResults: len(images),
	}
	for _, image := range images {
		res.Results = append(res.Results, registry.SearchResult{
			Name:        image.Name,
			Description: image.Description,
			IsAutomated: image.Is_automated,
			IsOfficial:  image.Is_official,
			StarCount:   int(image.Star_count),
		})
	}
	return
}
