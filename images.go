package dipod

import (
	"encoding/json"
	"errors"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/EricHripko/dipod/iopodman"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/pkg/jsonmessage"
	"github.com/docker/go-connections/nat"
	"github.com/gorilla/mux"
	"github.com/moby/moby/api/types"
	log "github.com/sirupsen/logrus"
	"github.com/varlink/go/varlink"
)

// ImageList is a handler function for /images/json.
func ImageList(res http.ResponseWriter, req *http.Request) {
	all := req.FormValue("all")
	if all == "1" || all == "true" {
		WriteError(res, http.StatusNotImplemented, ErrNotImplemented)
		return
	}
	filters, err := filters.FromParam(req.FormValue("filters"))
	if err != nil {
		log.WithError(err).Warn("image list filter fail")
	}
	log.WithField("filters", filters).WithField("all", all).Debug("image list")

	srcs, _ := iopodman.ListImages().Call(podman)
	var imgs []types.ImageSummary
	for _, src := range srcs {
		if filters.Include("label") && !filters.MatchKVList("label", src.Labels) {
			continue
		}
		if filters.Include("reference") {
			matched := false
			for _, search := range filters.Get("reference") {
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

		img := types.ImageSummary{
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
		if img.RepoTags == nil {
			img.RepoTags = []string{"<none>:<none>"}
		}
		if img.RepoDigests == nil {
			img.RepoDigests = []string{"<none>@<none>"}
		}
		if created, err := time.Parse(time.RFC3339, src.Created); err == nil {
			img.Created = created.Unix()
		} else {
			log.
				WithError(err).
				WithField("created", src.Created).Warn("created parse fail")
		}

		imgs = append(imgs, img)
	}
	JSONResponse(res, imgs)
}

// ImageBuild is a handler function for /build.
func ImageBuild(res http.ResponseWriter, req *http.Request) {
	log.Debug("image build", req.RequestURI)

	// stash context tarball to a temp file
	ctx, err := ioutil.TempFile("", "dipod-build")
	if err != nil {
		StreamError(res, err)
		return
	}
	defer os.Remove(ctx.Name())
	io.Copy(ctx, req.Body)

	// parse request uri
	url, err := url.ParseRequestURI(req.RequestURI)
	if err != nil {
		StreamError(res, err)
		return
	}
	query := url.Query()
	in := iopodman.BuildInfo{
		Dockerfiles: []string{query.Get("dockerfile")},
		ContextDir:  ctx.Name(),
	}
	tags := query["t"]
	if len(tags) > 0 {
		in.Output = tags[0]
	}
	if len(tags) > 1 {
		in.AdditionalTags = tags[1:]
	}

	log.WithField("info", in).Debug("build")
	recv, err := iopodman.BuildImage().Send(podman, varlink.More, in)
	if err != nil {
		StreamError(res, err)
		log.
			WithField("err", ErrorMessage(err)).
			Error("image build fail")
	}
	flusher, hasFlusher := res.(http.Flusher)
	for {
		status, flags, err := recv()
		if err != nil {
			StreamError(res, err)
			log.
				WithField("err", ErrorMessage(err)).
				Error("image build fail")
		} else {
			for _, log := range status.Logs {
				msg := jsonmessage.JSONMessage{
					Stream: log,
					ID:     status.Id,
				}
				JSONResponse(res, msg)
			}
		}

		if hasFlusher {
			flusher.Flush()
		}
		if flags&varlink.Continues != varlink.Continues {
			break
		}
	}
}

// ImageCreate is a handler function for /images/create.
func ImageCreate(res http.ResponseWriter, req *http.Request) {
	var (
		fromImage = req.FormValue("fromImage")
		_         = req.FormValue("fromSrc")
		_         = req.FormValue("repo")
		tag       = req.FormValue("tag")
	)

	name := fromImage + ":" + tag
	// no slash => pulling from DockerHub, docker cli shadily strips docker.io/
	// prefix even if user explicitly specified it
	if !strings.ContainsAny(name, "/") {
		name = "docker.io/library/" + name
	}
	log.WithField("name", name).Debug("image pull")

	recv, err := iopodman.PullImage().Send(podman, varlink.More, name)
	if err != nil {
		StreamError(res, err)
		log.
			WithField("err", ErrorMessage(err)).
			Error("image pull fail")
		return
	}

	flusher, hasFlusher := res.(http.Flusher)
	for {
		status, flags, err := recv()
		if err != nil {
			StreamError(res, err)
			log.
				WithField("err", ErrorMessage(err)).
				Error("image pull fail")
		} else {
			for _, log := range status.Logs {
				msg := jsonmessage.JSONMessage{
					Stream: log,
					ID:     status.Id,
				}
				JSONResponse(res, msg)
			}
		}

		if hasFlusher {
			flusher.Flush()
		}
		if flags&varlink.Continues != varlink.Continues {
			break
		}
	}
}

var errImageName = errors.New("dipod: missing image name")
var errImageData = errors.New("dipod: cannot decode image data")

func is2ss(i []interface{}) (s []string) {
	for _, ii := range i {
		s = append(s, ii.(string))
	}
	return
}

// ImageInspect is a handler function for /images/{name}/json.
func ImageInspect(res http.ResponseWriter, req *http.Request) {
	vars := mux.Vars(req)
	name, ok := vars["name"]
	if !ok {
		log.WithError(errImageName).Error("image inspect fail")
		res.WriteHeader(http.StatusBadRequest)
		return
	}
	log := log.WithField("name", name)
	log.Debug("image inspect")

	// podman for some reason returns this as JSON string, need to decode
	payload, err := iopodman.InspectImage().Call(podman, name)
	if notFound, ok := err.(*iopodman.ImageNotFound); ok {
		WriteError(res, http.StatusNotFound, errors.New(notFound.Reason))
		return
	}
	if err != nil {
		WriteError(res, http.StatusInternalServerError, err)
		return
	}
	data := make(map[string]interface{})
	json.Unmarshal([]byte(payload), &data)

	digest := strings.TrimPrefix(data["Digest"].(string), "sha256:")
	image := types.ImageInspect{
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

	JSONResponse(res, image)
}
