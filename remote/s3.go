package remote

import (
  "bufio"
  "crypto/md5"
  "dogestry/client"
  "encoding/hex"
  "encoding/json"
  "github.com/lachie/goamz/aws"
  "github.com/lachie/goamz/s3"

  "fmt"
  "net/http"
  "net/url"
  "path"
  "path/filepath"
  "strings"

  "io"
  "os"
)

type S3Remote struct {
  BucketName string
  Bucket     *s3.Bucket
  KeyPrefix  string
  client     *s3.S3
}

var (
  S3DefaultRegion = "us-west-2"
)

func NewS3Remote(url url.URL) (*S3Remote, error) {
  s3, err := newS3Client(url)
  if err != nil {
    return &S3Remote{}, nil
  }

  prefix := strings.TrimPrefix(url.Path, "/")

  return &S3Remote{
    BucketName: url.Host,
    KeyPrefix:  prefix,
    client:     s3,
  }, nil
}

// create a new s3 client from the url
func newS3Client(url url.URL) (*s3.S3, error) {
  auth, err := getS3Auth()
  if err != nil {
    return &s3.S3{}, err
  }

  var regionName string
  regQuery := url.Query()["region"]
  if len(regQuery) > 0 && regQuery[0] != "" {
    regionName = regQuery[0]
  } else {
    regionName = S3DefaultRegion
  }

  region := aws.Regions[regionName]

  return s3.New(auth, region), nil
}

// determine the s3 auth from various sources
func getS3Auth() (aws.Auth, error) {
  //filepath.join(os.Getenv("HOME"), ".ec2", "")
  //os.Stat(

  return aws.GetAuth("", "")
}

// Remote: describe the remote
func (remote *S3Remote) Desc() string {
  return fmt.Sprintf("s3(bucket=%s, prefix=%s)", remote.BucketName, remote.KeyPrefix)
}

func (remote *S3Remote) Push(image, imageRoot string) error {
  remoteKeys, err := remote.repoKeys("")
  if err != nil {
    return err
  }

  localKeys, err := remote.localKeys(imageRoot)
  if err != nil {
    return err
  }

  for key, localKey := range localKeys {
    if remoteKey, ok := remoteKeys[key]; !ok || remoteKey.ETag != localKey.ETag {
      fmt.Println("pushing key", key)

      if err := remote.putFile(imageRoot, localKey.Key); err != nil {
        return err
      }
    }
  }

  return nil
}

func (remote *S3Remote) PullImageId(id, dst string) error {
  rootKey := "/images/" + id
  imageKeys, err := remote.repoKeys(rootKey)
  if err != nil {
    return err
  }

  return remote.getFiles(dst, rootKey, imageKeys)
}

func (remote *S3Remote) ParseTag(repo, tag string) (string, error) {
  bucket := remote.getBucket()

  file, err := bucket.Get(remote.tagFilePath(repo, tag))
  if s3err, ok := err.(*s3.Error); ok && s3err.StatusCode == 404 {
    // doesn't exist yet, deal with it
    return "", nil
  } else if err != nil {
    return "", err
  }

  return string(file), nil
}

func (remote *S3Remote) ResolveImageNameToId(image string) (string, error) {
  return ResolveImageNameToId(remote, image)
}

func (remote *S3Remote) ImageFullId(name string) (string, error) {
  remoteKeys, err := remote.repoKeys("/images")
  if err != nil {
    return "", err
  }

  for key, _ := range remoteKeys {
    parts := strings.Split(key, "/")
    if strings.HasPrefix(name, parts[0]) {
      return parts[0], nil
    }
  }

  return "", ErrNoSuchImage
}

func (remote *S3Remote) WalkImages(id string, walker ImageWalkFn) error {
  return WalkImages(remote, id, walker)
}

func (remote *S3Remote) ImageMetadata(id string) (client.Image, error) {
  jsonPath := path.Join(remote.imagePath(id), "json")
  image := client.Image{}

  imageJson, err := remote.getBucket().Get(jsonPath)
  if s3err, ok := err.(*s3.Error); ok && s3err.StatusCode == 404 {
    // doesn't exist yet, deal with it
    return image, ErrNoSuchImage
  } else if err != nil {
    return image, err
  }

  if err := json.Unmarshal(imageJson, &image); err != nil {
    return image, err
  }

  return image, nil
}

// get the configured bucket
func (remote *S3Remote) getBucket() *s3.Bucket {
  // memoise?
  return remote.client.Bucket(remote.BucketName)
}

// get repository keys from s3
func (remote *S3Remote) repoKeys(prefix string) (map[string]s3.Key, error) {
  repoKeys := make(map[string]s3.Key)

  cnt, err := remote.getBucket().GetBucketContentsWithPrefix(remote.KeyPrefix + prefix)
  if err != nil {
    return repoKeys, err
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

// Get repository keys from the local work dir.
// Returned as a map of s3.Key's for ease of comparison.
func (remote *S3Remote) localKeys(root string) (map[string]s3.Key, error) {
  localKeys := make(map[string]s3.Key)

  if root[len(root)-1] != '/' {
    root = root + "/"
  }

  err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
    if info.IsDir() {
      return nil
    }

    sum, err := md5File(path)
    if err != nil {
      return err
    }

    key := strings.TrimPrefix(path, root)

    localKeys[key] = s3.Key{
      Key:  key,
      ETag: sum,
    }

    return nil
  })

  // XXX hmmm
  if err != nil {
    return localKeys, nil
  }

  return localKeys, nil
}

// md5 file at path
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

// the full remote key (adds KeyPrefix)
func (remote *S3Remote) remoteKey(key string) string {
  return path.Join(remote.KeyPrefix, key)
}

// put a file with key from imageRoot to the s3 bucket
func (remote *S3Remote) putFile(imageRoot, key string) error {
  path := filepath.Join(imageRoot, key)
  key = remote.remoteKey(key)

  f, err := os.Open(path)
  if err != nil {
    return err
  }
  defer f.Close()

  finfo, err := os.Stat(path)
  if err != nil {
    return err
  }

  buff := bufio.NewReader(f)
  return remote.getBucket().PutReader(key, buff, finfo.Size(), "application/octet-stream", s3.Private)
}

// get files from the s3 bucket to a local path
func (remote *S3Remote) getFiles(dst, rootKey string, imageKeys map[string]s3.Key) error {
  for key, _ := range imageKeys {
    relKey := strings.TrimPrefix(key, rootKey)
    err := remote.getFile(filepath.Join(dst, relKey), key)
    if err != nil {
      return err
    }
  }

  return nil
}

// get a single file from the s3 bucket
func (remote *S3Remote) getFile(dst, key string) error {
  key = path.Join(remote.KeyPrefix, key)

  from, err := remote.getBucket().GetReader(key)
  if err != nil {
    return err
  }
  defer from.Close()
  bufFrom := bufio.NewReader(from)

  if err := os.MkdirAll(filepath.Dir(dst), 0700); err != nil {
    return err
  }

  to, err := os.Create(dst)
  if err != nil {
    return err
  }

  io.Copy(to, bufFrom)
  // TODO check if file exists
  return nil
}

// path to a tagfile
func (remote *S3Remote) tagFilePath(repo, tag string) string {
  return filepath.Join(remote.KeyPrefix, "repositories", repo, tag)
}

// path to an image dir
func (remote *S3Remote) imagePath(id string) string {
  return filepath.Join(remote.KeyPrefix, "images", id)
}
