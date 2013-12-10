package remote

import (
  "dogestry/client"
  "dogestry/config"
  "errors"
  "fmt"
  "net/url"
  "strings"
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
  Kind string
  Config config.Config
  Url url.URL
}


type ImageWalkFn func(id string, image client.Image, err error) error

type Remote interface {
  // push image and parent images to remote
  Push(image, imageRoot string) error

  // pull a single image from the remote
  PullImageId(id, imageRoot string) error

  // map repo:tag to id (like git rev-parse)
  ParseTag(repo, tag string) (string, error)

  // map a ref-like to id. "ref-like" could be a ref or an id.
  ResolveImageNameToId(image string) (string, error)

  ImageFullId(id string) (string, error)

  ImageMetadata(id string) (client.Image, error)

  // walk the image history on the remote, starting at id
  WalkImages(id string, walker ImageWalkFn) error

  // describe the remote
  Desc() string
}


func NewRemote(remote string, config config.Config) (Remote, error) {
  remoteConfig, err := resolveConfig(remote, config)
  if err != nil {
    return nil, err
  }

  fmt.Println("remcfg", remoteConfig)

  switch remoteConfig.Kind {
  case "local":
    return NewLocalRemote(remoteConfig)
  case "s3":
    return NewS3Remote(remoteConfig)
  default:
    return nil, fmt.Errorf("unknown remote type '%s'", remoteConfig.Kind)
  }
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
  remote,ok := config.Remote[remoteName]
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

func ResolveImageNameToId(remote Remote, image string) (string, error) {
  // first, try the repos
  repoName, repoTag := NormaliseImageName(image)
  if id, err := remote.ParseTag(repoName, repoTag); err != nil {
    return "", err
  } else if id != "" {
    return id, nil
  }

  // ok, no repo, search the images:
  fullId, err := remote.ImageFullId(image)
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
func WalkImages(remote Remote, id string, walker ImageWalkFn) error {
  if id == "" {
    return nil
  }

  img, err := remote.ImageMetadata(id)
  // image wasn't found
  if err != nil {
    return walker(id, client.Image{}, err)
  }

  err = walker(id, img, nil)
  if err != nil {
    // abort the walk
    if err == BreakWalk {
      return nil
    }
    return err
  }

  return remote.WalkImages(img.Parent, walker)
}
