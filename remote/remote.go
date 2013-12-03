package remote

import (
	"dogestry/client"
	"errors"
	"fmt"
	"net/url"
)

type Remote interface {
	Push(image, imageRoot string) error
	ResolveImageNameToId(image string) (string, error)
	WalkImages(id string, walker func(image client.Image) error) error
}

var (
	// ErrInvalidRemote is returned when the remote is not a valid.
	ErrInvalidRemote = errors.New("Invalid endpoint")
)

func NewRemote(remote string) (Remote, error) {
	remoteUrl, err := normaliseURL(remote)
	if err != nil {
		return nil, err
	}

	switch remoteUrl.Scheme {
	case "rsync":
		return NewRsyncRemote(*remoteUrl)
	default:
		return nil, fmt.Errorf("unknown remote type %s", remoteUrl.Scheme)
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
