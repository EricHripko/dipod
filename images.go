package dipod

import (
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"time"

	"github.com/EricHripko/dipod/iopodman"
	"github.com/docker/docker/pkg/jsonmessage"

	"github.com/moby/moby/api/types"
	log "github.com/sirupsen/logrus"
	"github.com/varlink/go/varlink"
)

// ImageList is a handler function for /images/json.
func ImageList(res http.ResponseWriter, req *http.Request) {
	log.Debug("image list")
	srcs, _ := iopodman.ListImages().Call(podman)
	var imgs []types.ImageSummary
	for _, src := range srcs {
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
	recv, _ := iopodman.BuildImage().Send(podman, varlink.More, in)
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
