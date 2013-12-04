package remote

import (
	"launchpad.net/goamz/aws"
	"launchpad.net/goamz/s3"
)

type S3Remote struct {
	Bucket    s3.Bucket
	KeyPrefix string
}

var (
  S3DefaultRegion = "us-west-2"
)

func NewS3Remote(url url.URL) (*S3Remote, error) {
  auth, err := aws.EnvAuth()
  if err != nil {
    return &S3Remote{}, err
  }

  regionName := url.Query()["region"]
  if regionName == "" {
    regionName = S3DefaultRegion
  }

  // hrmm
  auth, err := aws.EnvAuth()

  s3 := s3.New(auth, aws.Regions[regionName])

  bucket := s3.Bucket(url.Host)

	return &S3Remote{
    Bucket: bucket,
    KeyPrefix: url.Path,
  }, nil
}

func (remote *S3Remote) Desc() string {
}

func (remote *S3Remote) Push(image, imageRoot string) error {
}

func (remote *S3Remote) PullImageId(id, imageRoot string) error {
}

func (remote *S3Remote) ParseTag(repo, tag string) (string, error) {
}

func (remote *S3Remote) ResolveImageNameToId(image string) (string, error) {
}

func (remote *S3Remote) WalkImages(id string, walker ImageWalkFn) error {
}
