package remote

import (
	"github.com/lachie/aws4"
  "fmt"
  //"io/ioutil"
  "net/http"
  "net/url"
  "time"

  "io"
  "os"
)

type S3Remote struct {
	Bucket    string
	KeyPrefix string
  client aws4.Client
}

var (
	S3DefaultRegion = "us-west-2"
)

func redirectPolicyFunc(req *http.Request, via []*http.Request) error {
  return fmt.Errorf("no redirects")
}

func NewS3Remote(url url.URL) (*S3Remote, error) {

	c, err := aws4.NewClientFromEnv()
	if err != nil {
		return &S3Remote{}, err
	}

  c.Client = &http.Client{
    CheckRedirect: redirectPolicyFunc,
  }


	return &S3Remote{
		Bucket:    url.Host,
		KeyPrefix: url.Path,
    client: c,
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
  bucket,err := remote.getBucket()
  if err != nil {
    return "", err
  }
  fmt.Println("b", bucket)
  return "", nil
}

func (remote *S3Remote) ResolveImageNameToId(image string) (string, error) {
  return "", nil
}

func (remote *S3Remote) WalkImages(id string, walker ImageWalkFn) error {
  return nil
}


func (remote *S3Remote) getBucket() (*S3Bucket, error) {
  url := "https://s3-us-west-2.amazonaws.com/"

  fmt.Println("url", url)

  r, _ := http.NewRequest("GET", url, nil)
  r.Header.Set("Host", remote.Bucket+".s3-us-west-2.amazonaws.com")
  r.Header.Set("Date", time.Now().Format(http.TimeFormat))

  fmt.Println("r", r)

  resp,err := remote.client.Do(r)
  if err != nil {
    fmt.Println("err Do", resp)
    io.Copy(os.Stdout, resp.Body)
    return &S3Bucket{}, err
  }
  if resp.StatusCode != 200 {
    return &S3Bucket{}, fmt.Errorf("error getting bucket location: %s", resp.Status)
  }

  //io.Copy(os.Stdout, resp.Body)

  fmt.Println("resp", resp)

  return &S3Bucket{}, nil
}


type S3Bucket struct {
  Name string
}
