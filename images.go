package dipod

import (
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/EricHripko/dipod/iopodman"
	"github.com/docker/docker/pkg/jsonmessage"

	"github.com/docker/docker/api/types/filters"
	"github.com/moby/moby/api/types"
	log "github.com/sirupsen/logrus"
	"github.com/varlink/go/varlink"
)

// ImageList is a handler function for /images/json.
func ImageList(res http.ResponseWriter, req *http.Request) {
	all := req.FormValue("all")
	if all == "1" || all == "true" {
		WriteError(res, ErrNotImplemented)
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
		name = "docker.io/" + name
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
