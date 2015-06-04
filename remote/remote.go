package remote

import (
	"errors"
	"strings"

	"github.com/dogestry/dogestry/config"
	docker "github.com/fsouza/go-dockerclient"
)

var (
	ErrInvalidRemote = errors.New("Invalid remote")

	ErrNoSuchImage = errors.New("No such image")
	ErrNoSuchTag   = errors.New("No such tag")
	BreakWalk      = errors.New("break walk")
)

type Image struct {
	Repository string
	Tag        string
}

type ImageWalkFn func(id ID, image docker.Image, err error) error

type Remote interface {
	// push image and parent images to remote
	Push(image, imageRoot string) error

	// pull a single image from the remote
	PullImageId(id ID, imageRoot string) error

	// map repo:tag to id (like git rev-parse)
	ParseTag(repo, tag string) (ID, error)

	// map a ref-like to id. "ref-like" could be a ref or an id.
	ResolveImageNameToId(image string) (ID, error)

	ImageFullId(id ID) (ID, error)

	ImageMetadata(id ID) (docker.Image, error)

	// return repo, tag from a file path (or S3 key)
	ParseImagePath(path string, prefix string) (repo, tag string)

	// walk the image history on the remote, starting at id
	WalkImages(id ID, walker ImageWalkFn) error

	// checks the config and connectivity of the remote
	Validate() error

	// describe the remote
	Desc() string

	// List images on the remote
	List() ([]Image, error)
}

func NewRemote(config config.Config) (Remote, error) {
	remote, err := NewS3Remote(config)
	if err != nil {
		return nil, err
	}

	err = remote.Validate()
	if err != nil {
		return nil, err
	}

	return remote, nil
}

func NormaliseImageName(image string) (string, string) {
	repoParts := strings.Split(image, ":")
	if len(repoParts) == 1 {
		return repoParts[0], "latest"
	} else {
		return repoParts[0], repoParts[1]
	}
}

func ResolveImageNameToId(remote Remote, image string) (ID, error) {
	// first, try the repos
	repoName, repoTag := NormaliseImageName(image)
	if id, err := remote.ParseTag(repoName, repoTag); err != nil {
		return "", err
	} else if id != "" {
		return id, nil
	}

	// ok, no repo, search the images:
	fullId, err := remote.ImageFullId(ID(image))
	if err != nil {
		return "", err
	} else if fullId != "" {
		return fullId, nil
	}

	return "", ErrNoSuchImage
}

func ParseImagePath(path string, prefix string) (repo, tag string) {
	path = strings.TrimPrefix(path, prefix)
	parts := strings.Split(path, "/")
	repo = strings.Join(parts[:len(parts)-1], "/")
	tag = parts[len(parts)-1]
	return repo, tag
}

// Common implementation of walking a remote's images
//
// Starting at id, follow the ancestry tree, calling walker for each image found.
// Walker can abort the walk by returning an error.
// - BreakWalk - the walk stops and WalkImages returns nil (no error)
// - other error - the walk stop and WalkImages returns the error.
// - nil - the walk continues
func WalkImages(remote Remote, id ID, walker ImageWalkFn) error {
	if id == "" {
		return nil
	}

	img, err := remote.ImageMetadata(id)
	// image wasn't found
	if err != nil {
		return walker(id, docker.Image{}, err)
	}

	err = walker(id, img, nil)
	if err != nil {
		// abort the walk
		if err == BreakWalk {
			return nil
		}
		return err
	}

	return remote.WalkImages(ID(img.Parent), walker)
}
