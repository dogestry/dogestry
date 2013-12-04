package remote

import (
	"launchpad.net/goamz/s3"
)

type S3Remote struct {
	Bucket    string
	KeyPrefix string
}

func NewS3Remote(url url.URL) (*S3Remote, error) {
	return &S3Remote{}
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
