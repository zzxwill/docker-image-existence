package image

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/heroku/docker-registry-client/registry"
)

type Meta struct {
	Registry   string
	Repository string
	Name       string
	Tag        string
}

type DockerHubImageTagResponse struct {
	Count   int `json:"count"`
	Results []Result
}

type Result struct {
	Name string `json:"name"`
}

func IsExisted(username, password, image string) (bool, error) {
	meta, err := retrieveImageMeta(image)
	if err != nil {
		return false, err
	}

	if username != "" || password != "" {
		hub, err := registry.New(meta.Registry, username, password)
		if err != nil {
			return false, err
		}
		digest, err := hub.ManifestDigest(meta.Repository+"/"+meta.Name, meta.Tag)
		if err != nil {
			return false, err
		}
		if digest == "" {
			return false, fmt.Errorf("image %s not found as its degest is empty", image)
		}
		return true, nil
	}

	switch meta.Registry {
	case "hub.docker.com":
		api := fmt.Sprintf("https://%s/v2/repositories/%s/%s/tags?page_size=10000", meta.Registry, meta.Repository, meta.Name)
		resp, err := http.Get(api)
		if err != nil {
			return false, err
		}
		if resp.StatusCode == 200 {
			var r DockerHubImageTagResponse
			var tagExisted bool
			body, err := io.ReadAll(resp.Body)
			if err != nil {
				return false, err
			}
			if err := json.Unmarshal(body, &r); err == nil {
				for _, result := range r.Results {
					if result.Name == meta.Tag {
						tagExisted = true
						break
					}
				}
			}
			if tagExisted {
				return true, nil
			}
			return false, fmt.Errorf("image %s not found as its tag %s is not existed", meta.Name, meta.Tag)
		}
		return false, nil
	default:
		return false, fmt.Errorf("image doesn't exist as its registry %s is not supported yet", meta.Registry)
	}
}

func retrieveImageMeta(image string) (*Meta, error) {
	var (
		reg  string
		repo string
		name string
		tag  string
	)
	if image == "" {
		return nil, fmt.Errorf("image is empty")
	}
	meta := strings.Split(image, ":")
	if len(meta) == 1 {
		tag = "latest"
	} else {
		tag = meta[1]
	}

	tmp := strings.Split(meta[0], "/")
	switch len(tmp) {
	case 1:
		reg = "hub.docker.com"
		repo = "library"
		name = tmp[0]
	case 2:
		if tmp[0] == "docker.io" {
			reg = "hub.docker.com"
			repo = "library"
		} else {
			reg = "hub.docker.com"
			repo = tmp[0]
		}
		name = tmp[1]
	case 3:
		if tmp[0] == "docker.io" {
			reg = "hub.docker.com"
		} else {
			reg = tmp[0]
		}
		repo = tmp[1]
		name = tmp[2]
	}
	return &Meta{Registry: reg, Repository: repo, Name: name, Tag: tag}, nil
}
