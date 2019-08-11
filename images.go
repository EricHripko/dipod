package dipod

import (
	"encoding/json"
	"errors"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/EricHripko/dipod/iopodman"

	"github.com/docker/distribution/reference"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/api/types/registry"
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
		WriteError(res, http.StatusBadRequest, errImageName)
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

// ImageHistory is a handler function for /images/{name}/history.
func ImageHistory(res http.ResponseWriter, req *http.Request) {
	vars := mux.Vars(req)
	name, ok := vars["name"]
	if !ok {
		log.WithError(errImageName).Error("image history fail")
		WriteError(res, http.StatusBadRequest, errImageName)
		return
	}
	log := log.WithField("name", name)
	log.Debug("image history")

	var history []types.ImageHistory
	backend, err := iopodman.HistoryImage().Call(podman, name)
	if notFound, ok := err.(*iopodman.ImageNotFound); ok {
		WriteError(res, http.StatusNotFound, errors.New(notFound.Reason))
		return
	}
	if err != nil {
		WriteError(res, http.StatusInternalServerError, err)
		return
	}

	for _, l := range backend {
		layer := types.ImageHistory{
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

		history = append(history, layer)
	}
	JSONResponse(res, history)
}

// ImageTag is a handler function for /images/{name}/tag.
func ImageTag(res http.ResponseWriter, req *http.Request) {
	vars := mux.Vars(req)
	source, ok := vars["name"]
	if !ok {
		log.WithError(errImageName).Error("image tag fail")
		WriteError(res, http.StatusBadRequest, errImageName)
		return
	}
	target := req.FormValue("repo")
	tag := req.FormValue("tag")
	if tag != "" {
		target += ":" + tag
	}
	log := log.WithField("source", source).WithField("target", target)
	if target == "" {
		err := errors.New("dipod: empty target")
		log.WithError(err).Error("image tag fail")
		WriteError(res, http.StatusBadRequest, errImageName)
	}
	log.Debug("image tag")

	_, err := iopodman.TagImage().Call(podman, source, target)
	if notFound, ok := err.(*iopodman.ImageNotFound); ok {
		WriteError(res, http.StatusNotFound, errors.New(notFound.Reason))
		return
	}
	if err != nil {
		WriteError(res, http.StatusInternalServerError, err)
		return
	}
	res.WriteHeader(http.StatusNoContent)
}

// ImageDelete is a handler function for /images/{name}.
func ImageDelete(res http.ResponseWriter, req *http.Request) {
	vars := mux.Vars(req)
	name, ok := vars["name"]
	if !ok {
		log.WithError(errImageName).Error("image tag fail")
		WriteError(res, http.StatusBadRequest, errImageName)
		return
	}
	force := req.FormValue("force")
	log := log.WithField("name", name).WithField("force", force)
	log.Debug("image delete")

	deleted, err := iopodman.RemoveImage().Call(
		podman,
		name,
		force == "true" || force == "1",
	)
	if notFound, ok := err.(*iopodman.ImageNotFound); ok {
		WriteError(res, http.StatusNotFound, errors.New(notFound.Reason))
		return
	}
	if err != nil {
		WriteError(res, http.StatusInternalServerError, err)
		return
	}

	JSONResponse(res, []types.ImageDelete{types.ImageDelete{Deleted: deleted}})
}

// ImageSearch is a handler function for /images/search.
func ImageSearch(res http.ResponseWriter, req *http.Request) {
	query := req.FormValue("term")
	if query == "" {
		err := errors.New("dipod: missing term")
		log.WithError(err).Error("image tag fail")
		WriteError(res, http.StatusBadRequest, err)
		return
	}
	var limit *int64
	sLimit := req.FormValue("limit")
	if sLimit != "" {
		nLimit, err := strconv.Atoi(sLimit)
		var lLimit int64
		if err != nil {
			log.
				WithError(err).
				WithField("limit", sLimit).
				Warn("image search ignore invalid limit")
		} else {
			lLimit = int64(nLimit)
			limit = &lLimit
		}
	}
	filters, err := filters.FromParam(req.FormValue("filters"))
	if err != nil {
		log.WithError(err).Warn("image search filter fail")
	}
	log.
		WithFields(log.Fields{
			"query":   query,
			"limit":   sLimit,
			"filters": filters,
		}).
		Debug("image search")

	yes := true
	no := false
	filter := iopodman.ImageSearchFilter{}
	isAutomated := filters.Get("is-automated")
	if len(isAutomated) > 0 && (isAutomated[0] == "true" || isAutomated[0] == "1") {
		filter.Is_automated = &yes
	}
	if len(isAutomated) > 0 && (isAutomated[0] == "false" || isAutomated[0] == "0") {
		filter.Is_automated = &no
	}
	isOfficial := filters.Get("is-official")
	if len(isOfficial) > 0 && (isOfficial[0] == "true" || isOfficial[0] == "1") {
		filter.Is_official = &yes
	}
	if len(isOfficial) > 0 && (isOfficial[0] == "false" || isOfficial[0] == "0") {
		filter.Is_official = &no
	}
	stars := filters.Get("stars")
	if len(stars) > 0 {
		nStars, err := strconv.Atoi(stars[0])
		if err != nil {
			log.WithError(err).Warn("image search star filter fail")
		} else {
			filter.Star_count = int64(nStars)
		}
	}

	srcs, err := iopodman.SearchImages().Call(podman, query, limit, filter)
	if err != nil {
		WriteError(res, http.StatusInternalServerError, err)
		return
	}

	var images []registry.SearchResult
	for _, src := range srcs {
		images = append(images, registry.SearchResult{
			Name:        src.Name,
			Description: src.Description,
			IsAutomated: src.Is_automated,
			IsOfficial:  src.Is_official,
			StarCount:   int(src.Star_count),
		})
	}
	JSONResponse(res, images)
}

func exportImages(res http.ResponseWriter, names []string, log *log.Entry) {
	// prepare temp file for the tarball
	tmp, err := ioutil.TempFile("", "dipod-export")
	if err != nil {
		log.WithError(err).Error("image export fail")
		WriteError(res, http.StatusInternalServerError, err)
		return
	}
	dest := "docker-archive://" + tmp.Name()
	err = tmp.Close()
	if err != nil {
		log.WithError(err).Error("image export fail")
		WriteError(res, http.StatusInternalServerError, err)
		return
	}
	defer os.Remove(tmp.Name())

	// parse list of image names into name + list of tags
	ref, err := reference.Parse(names[0])
	if err != nil {
		log.WithError(err).Error("image export fail")
		WriteError(res, http.StatusBadRequest, err)
		return
	}
	named, ok := ref.(reference.Named)
	if !ok {
		err := errors.New("dipod: main name parse fail")
		log.WithError(err).Error("image export fail")
		WriteError(res, http.StatusBadRequest, err)
		return
	}
	var tags []string
	for _, name := range names[1:] {
		ref, err := reference.Parse(name)
		if err != nil {
			log.WithError(err).Error("image export fail")
			WriteError(res, http.StatusBadRequest, err)
			return
		}
		nt, ok := ref.(reference.NamedTagged)
		if !ok {
			err := errors.New("dipod: secondary name parse fail")
			log.WithError(err).Error("image export fail")
			WriteError(res, http.StatusBadRequest, err)
			return
		}
		if named.Name() != nt.Name() {
			err := errors.New("dipod: multiple image export not supported")
			log.WithError(err).Error("image export fail")
			WriteError(res, http.StatusNotImplemented, err)
			return
		}
		tags = append(tags, nt.Tag())
	}

	_, err = iopodman.ExportImage().Call(podman, names[0], dest, false, tags)
	if notFound, ok := err.(*iopodman.ImageNotFound); ok {
		WriteError(res, http.StatusNotFound, errors.New(notFound.Reason))
		return
	}
	if err != nil {
		log.WithError(err).Error("image export fail")
		WriteError(res, http.StatusInternalServerError, err)
		return
	}

	tmp, err = os.Open(tmp.Name())
	if err != nil {
		log.WithError(err).Error("image export fail")
		WriteError(res, http.StatusInternalServerError, err)
		return
	}

	res.Header().Set("Content-Type", "application/x-tar")
	io.Copy(res, tmp)
}

// ImageGet is a handler function for /images/{name}/get.
func ImageGet(res http.ResponseWriter, req *http.Request) {
	vars := mux.Vars(req)
	name, ok := vars["name"]
	if !ok {
		log.WithError(errImageName).Error("image export fail")
		WriteError(res, http.StatusBadRequest, errImageName)
		return
	}
	log := log.WithField("name", name)
	log.Debug("image export")

	exportImages(res, []string{name}, log)
}

// ImageGetAll is a handler function for /images/get.
func ImageGetAll(res http.ResponseWriter, req *http.Request) {
	err := req.ParseForm()
	if err != nil {
		log.WithError(err).Error("image export fail")
		WriteError(res, http.StatusInternalServerError, err)
		return
	}
	names := req.Form["names"]
	if len(names) == 0 {
		log.WithError(err).Error("image export fail")
		WriteError(res, http.StatusInternalServerError, err)
		return
	}
	log := log.WithField("names", names)
	log.Debug("image bulk export")

	exportImages(res, names, log)
}
