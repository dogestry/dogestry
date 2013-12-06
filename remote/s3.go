package remote

import (
	"launchpad.net/goamz/s3"
	"launchpad.net/goamz/aws"

  "fmt"
  //"io/ioutil"
  "net/http"
  "net/url"
  //"time"
  "path/filepath"

  //"io"
  //"os"
)

type S3Remote struct {
  BucketName string
	Bucket    *s3.Bucket
	KeyPrefix string
  client *s3.S3
}

var (
	S3DefaultRegion = "us-west-2"
)

func redirectPolicyFunc(req *http.Request, via []*http.Request) error {
  return fmt.Errorf("no redirects")
}

func NewS3Remote(url url.URL) (*S3Remote, error) {
  auth, err := aws.EnvAuth()
  if err != nil {
    return nil, err
  }

  s3 := s3.New(auth, aws.USWest2)


	return &S3Remote{
		BucketName:    url.Host,
		KeyPrefix: url.Path,
    client: s3,
	}, nil
}

func (remote *S3Remote) Desc() string {
  return fmt.Sprintf("s3(bucket=%s, prefix=%s)", remote.Bucket, remote.KeyPrefix)
}

func (remote *S3Remote) Push(image, imageRoot string) error {
  return nil
}

func (remote *S3Remote) PullImageId(id, imageRoot string) error {
  return nil
}

func (remote *S3Remote) ParseTag(repo, tag string) (string, error) {
  bucket := remote.getBucket()
  fmt.Println("b", bucket)

  file,err := bucket.Get(TagFilePath(repo, tag))
  if s3err,ok := err.(*s3.Error); ok && s3err.StatusCode == 404 {
    // doesn't exist yet, deal with it
    return "", nil
  } else if err != nil {
    return "", err
  }

  fmt.Println("got", string(file))

  return string(file), nil
}

func (remote *S3Remote) ResolveImageNameToId(image string) (string, error) {
  return "", nil
}

func (remote *S3Remote) WalkImages(id string, walker ImageWalkFn) error {
  return nil
}


func (remote *S3Remote) getBucket() (*s3.Bucket) {
  return remote.client.Bucket(remote.BucketName)
}


type S3Bucket struct {
  Name string
}


func TagFilePath(repo, tag string) string {
  return filepath.Join("repositories", repo, tag)
}
