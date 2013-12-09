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
  // push image and parent images to remote
  Push(image, imageRoot string) error

  // pull a single image from the remote
  PullImageId(id, imageRoot string) error

  // map repo:tag to id (like git rev-parse)
  ParseTag(repo, tag string) (string, error)

  // map a ref-like to id. "ref-like" could be a ref or an id.
  ResolveImageNameToId(image string) (string, error)

  ImageFullId(id string) (string,error)

  // walk the image history on the remote, starting at id
  WalkImages(id string, walker ImageWalkFn) error

  // describe the remote
  Desc() string
}


type AbstractRemote struct {}

func NewRemote(remote string) (Remote, error) {
  remoteUrl, err := normaliseURL(remote)
  if err != nil {
    return nil, err
  }

  switch remoteUrl.Scheme {
  case "local":
    return NewLocalRemote(*remoteUrl)
  case "s3":
    return NewS3Remote(*remoteUrl)
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



func (remote Remote) ResolveImageNameToId(image string) (string, error) {
  fmt.Println("hi resolving")

  // first, try the repos
  repoName, repoTag := NormaliseImageName(image)
  if id, err := remote.ParseTag(repoName, repoTag); err != nil {
    return "", err
  } else if id != "" {
    return id, nil
  }


  // ok, no repo, search the images:
  fullId,err := remote.ImageFullId(image)
  if err != nil {
    return "", err
  } else if found {
    return fullId, nil
  }

  return "", ErrNoSuchImage
}
