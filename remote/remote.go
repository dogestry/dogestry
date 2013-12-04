package remote

import (
	"dogestry/client"
	"errors"
	"fmt"
	"net/url"
	"strings"
)

var (
	// ErrInvalidRemote is returned when the remote is not a valid.
	ErrInvalidRemote = errors.New("Invalid endpoint")

	ErrNoSuchImage = errors.New("No such image")
	ErrNoSuchTag   = errors.New("No such tag")
	BreakWalk      = errors.New("break walk")
)

type ImageWalkFn func(id string, image client.Image, err error) error

type Remote interface {
	ParseTag(repo, tag string) (string, error)
	Push(image, imageRoot string) error
	PullImageId(id, imageRoot string) error
	ResolveImageNameToId(image string) (string, error)
	WalkImages(id string, walker ImageWalkFn) error
	Desc() string
}

func NewRemote(remote string) (Remote, error) {
	remoteUrl, err := normaliseURL(remote)
	if err != nil {
		return nil, err
	}

	switch remoteUrl.Scheme {
	case "local":
		return NewLocalRemote(*remoteUrl)
	default:
		return nil, fmt.Errorf("unknown remote type %s", remoteUrl.Scheme)
	}
}

func normaliseURL(remoteUrl string) (*url.URL, error) {
	fmt.Println("url in", remoteUrl)
	u, err := url.Parse(remoteUrl)
	if err != nil {
		return nil, ErrInvalidRemote
	}

	if u.Scheme == "" {
		u.Scheme = "local"
	}

	fmt.Println("sch", u.Scheme, u.Path)

	return u, nil
}

func NormaliseImageName(image string) (string, string) {
	repoParts := strings.Split(image, ":")
	if len(repoParts) == 1 {
		return repoParts[0], "latest"
	} else {
		return repoParts[0], repoParts[1]
	}
}
