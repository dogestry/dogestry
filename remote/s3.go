package remote

import (
	"github.com/lachie/goamz/s3"
	"github.com/lachie/goamz/aws"
  "crypto/md5"
  "encoding/hex"
  "bufio"

  "fmt"
  //"io/ioutil"
  "net/http"
  "net/url"
  //"time"
  "path/filepath"
  "path"
  "strings"

  "io"
  "os"
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

  prefix := strings.TrimPrefix(url.Path, "/")


	return &S3Remote{
		BucketName:    url.Host,
		KeyPrefix: prefix,
    client: s3,
	}, nil
}

func (remote *S3Remote) Desc() string {
  return fmt.Sprintf("s3(bucket=%s, prefix=%s)", remote.BucketName, remote.KeyPrefix)
}

func (remote *S3Remote) Push(image, imageRoot string) error {
  remoteKeys,err := remote.repoKeys()
  if err != nil {
    return err
  }

  localKeys,err := remote.localKeys(imageRoot)
  if err != nil {
    return err
  }

  for name, key := range localKeys {
    fmt.Println("local name", name, "etag", key.ETag)
  }

  for name, key := range remoteKeys {
    fmt.Println("   s3 name", name, "etag", key.ETag)
  }

  for key,localKey := range localKeys {
    if remoteKey,ok := remoteKeys[key]; !ok || remoteKey.ETag != localKey.ETag {
      fmt.Println("want to push", key)

      if err := remote.putFile(imageRoot, localKey.Key); err != nil {
        return err
      }
    }
  }

  return nil
}



func (remote *S3Remote) PullImageId(id, imageRoot string) error {
  return nil
}

func (remote *S3Remote) ParseTag(repo, tag string) (string, error) {
  bucket := remote.getBucket()

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
  // memoise?
  return remote.client.Bucket(remote.BucketName)
}


type S3Bucket struct {
  Name string
}


func TagFilePath(repo, tag string) string {
  return filepath.Join("repositories", repo, tag)
}


func (remote *S3Remote) repoKeys() (map[string]s3.Key, error) {
  repoKeys := make(map[string]s3.Key)

  cnt,err := remote.getBucket().GetBucketContentsWithPrefix(remote.KeyPrefix)
  if err != nil {
    return repoKeys,err
  }

  for name, key := range *cnt {
    key.Key = strings.TrimPrefix(name, remote.KeyPrefix)
    key.ETag = strings.TrimRight(strings.TrimLeft(key.ETag, "\""), "\"")
    if key.Key != "" {
      repoKeys[key.Key] = key
    }
  }

  return repoKeys, nil
}


func (remote *S3Remote) localKeys(root string) (map[string]s3.Key, error) {
  repoKeys := make(map[string]s3.Key)

  if root[len(root)-1] != '/' {
    root = root + "/"
  }

  err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
    if info.IsDir() {
      return nil
    }

    sum,err := md5File(path)
    if err != nil {
      return err
    }

    key := strings.TrimPrefix(path, root)

    repoKeys[key] = s3.Key{
      Key: key,
      ETag: sum,
    }

    return nil
  })

  if err != nil {
    return repoKeys, nil
  }

  return repoKeys, nil
}

func md5File(path string) (string, error) {
  f, err := os.Open(path)
  if err != nil {
    return "", nil
  }
  defer f.Close()

  // files could be pretty big, lets buffer
  buff := bufio.NewReader(f)
  hash := md5.New()

  io.Copy(hash, buff)
  return hex.EncodeToString(hash.Sum(nil)), nil
}


func (remote *S3Remote) remoteKey(key string) string {
  return path.Join(remote.KeyPrefix, key)
}


func (remote *S3Remote) putFile(imageRoot, key string) error {
  path := filepath.Join(imageRoot, key)
  key = remote.remoteKey(key)

  f,err := os.Open(path)
  if err != nil {
    return err
  }
  defer f.Close()

  finfo,err := os.Stat(path)
  if err != nil {
    return err
  }

  fmt.Println("putting", key, finfo.Size())

  buff := bufio.NewReader(f)
  return remote.getBucket().PutReader(key, buff, finfo.Size(), "application/octet-stream", s3.Private)
}
