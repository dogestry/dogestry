package remote

import (
	"errors"
	"fmt"
	"net/url"
	"strings"

	"github.com/blake-education/dogestry/config"
	docker "github.com/fsouza/go-dockerclient"
)

var (
	// ErrInvalidRemote is returned when the remote is not a valid.
	ErrInvalidRemote = errors.New("Invalid remote")

	ErrNoSuchImage = errors.New("No such image")
	ErrNoSuchTag   = errors.New("No such tag")
	BreakWalk      = errors.New("break walk")
)

type RemoteConfig struct {
	config.RemoteConfig
	Kind   string
	Config config.Config
	Url    url.URL
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

	// walk the image history on the remote, starting at id
	WalkImages(id ID, walker ImageWalkFn) error

	// checks the config and connectivity of the remote
	Validate() error

	// describe the remote
	Desc() string
}

func NewRemote(remoteName string, config config.Config) (remote Remote, err error) {
	remoteConfig, err := resolveConfig(remoteName, config)
	if err != nil {
		return
	}

	switch remoteConfig.Kind {
	case "local":
		remote, err = NewLocalRemote(remoteConfig)
	case "s3":
		remote, err = NewS3Remote(remoteConfig)
	default:
		err = fmt.Errorf("unknown remote type '%s'", remoteConfig.Kind)
		return
	}

	if err != nil {
		return
	}

	err = remote.Validate()
	return
}

func resolveConfig(remoteUrl string, config config.Config) (remoteConfig RemoteConfig, err error) {
	// its a bareword, use it as a lookup key
	if !strings.Contains(remoteUrl, "/") {
		return lookupUrlInConfig(remoteUrl, config)
	}

	// its a url
	return makeRemoteFromUrl(remoteUrl, config)
}

func lookupUrlInConfig(remoteName string, config config.Config) (remoteConfig RemoteConfig, err error) {
	remote, ok := config.Remote[remoteName]
	if !ok {
		err = fmt.Errorf("no remote '%s' found", remoteName)
		return
	}

	return makeRemoteFromUrl(remote.Url, config)
	// XXX Extra setup can come from here
}

func makeRemoteFromUrl(remoteUrl string, config config.Config) (remoteConfig RemoteConfig, err error) {
	remoteConfig = RemoteConfig{
		Config: config,
	}

	u, err := url.Parse(remoteUrl)
	if err != nil {
		err = ErrInvalidRemote
		return
	}

	if u.Scheme == "" {
		u.Scheme = "local"
	}
	remoteConfig.Url = *u
	remoteConfig.Kind = u.Scheme
	remoteConfig.Config = config

	return
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
