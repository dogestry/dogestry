package remote

import (
	"errors"
	"fmt"
	"net/url"
)

type Remote interface {
	Push() error
	ResolveImageName(image string) (string, error)
}

type RemoteSpec struct {
	url       url.URL
	image     string
	imageRoot string
}

var (
	// ErrInvalidRemote is returned when the remote is not a valid.
	ErrInvalidRemote = errors.New("Invalid endpoint")
)

func Push(remote, image, imageRoot string) error {
	r, err := findRemoteImpl(remote, image, imageRoot)
	if err != nil {
		return err
	}

	return r.Push()
}

func ResolveImageName(remote, image string) (string, error) {
	r, err := findRemoteImpl(remote, image, "")
	if err != nil {
		return "", err
	}

	return r.ResolveImageName(image)
}

func findRemoteImpl(remote, image, imageRoot string) (Remote, error) {
	remoteUrl, err := normaliseURL(remote)
	if err != nil {
		return nil, err
	}

	spec := RemoteSpec{
		url:       *remoteUrl,
		image:     image,
		imageRoot: imageRoot,
	}

	switch spec.url.Scheme {
	case "rsync":
		return NewRsyncRemote(spec)
	default:
		return nil, fmt.Errorf("unknown remote type %s", spec.url.Scheme)
	}
}

func normaliseURL(remoteUrl string) (*url.URL, error) {
	u, err := url.Parse(remoteUrl)
	if err != nil {
		return nil, ErrInvalidRemote
	}

	if u.Scheme == "" {
		u.Scheme = "rsync"
	}

	fmt.Println("sch", u.Scheme, u.Path)

	return u, nil
}
