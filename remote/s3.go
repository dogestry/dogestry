package remote

import (
  "github.com/lachie/goamz/aws"
  "github.com/lachie/goamz/s3"
  "dogestry/utils"

  "bufio"
  "crypto/md5"
  "dogestry/client"
  "encoding/hex"
  "encoding/json"


  "fmt"
  "path"
  "path/filepath"
  "strings"

  "io"
  "os"
)

type S3Remote struct {
  config RemoteConfig
  BucketName string
  Bucket     *s3.Bucket
  KeyPrefix  string
  client     *s3.S3
}

var (
  S3DefaultRegion = "us-west-2"
)

func NewS3Remote(config RemoteConfig) (*S3Remote, error) {
  s3, err := newS3Client(config)
  if err != nil {
    return &S3Remote{}, nil
  }

  url := config.Url
  prefix := strings.TrimPrefix(url.Path, "/")

  return &S3Remote{
    config: config,
    BucketName: url.Host,
    KeyPrefix:  prefix,
    client:     s3,
  }, nil
}

// create a new s3 client from the url
func newS3Client(config RemoteConfig) (*s3.S3, error) {
  auth, err := getS3Auth(config)
  if err != nil {
    return &s3.S3{}, err
  }

  var regionName string
  regQuery := config.Url.Query()["region"]
  if len(regQuery) > 0 && regQuery[0] != "" {
    regionName = regQuery[0]
  } else {
    // TODO get default region from config
    regionName = S3DefaultRegion
  }

  region := aws.Regions[regionName]

  return s3.New(auth, region), nil
}

// determine the s3 auth from various sources
func getS3Auth(config RemoteConfig) (aws.Auth, error) {
  s3config := config.Config.S3
  return aws.GetAuth(s3config.Access_Key_Id, s3config.Secret_Key)
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

  // DEBUG
  delete(remoteKeys, "images/8dbd9e392a964056420e5d58ca5cc376ef18e2de93b5cc90e868a1bbc8318c1c/layer.tar.lz4")

  for key, localKey := range localKeys {

    if remoteKey, ok := remoteKeys[key]; !ok || remoteKey.ETag != localKey.ETag {
      fmt.Printf("pushing key %s (%s)\n", key, utils.FileHumanSize(filepath.Join(imageRoot,localKey.Key)))

      if err := remote.putFile(imageRoot, localKey.Key); err != nil {
        return err
      }
    }
  }

  return nil
}

func (remote *S3Remote) PullImageId(id ID, dst string) error {
  rootKey := "/images/" + string(id)
  imageKeys, err := remote.repoKeys(rootKey)
  if err != nil {
    return err
  }

  return remote.getFiles(dst, rootKey, imageKeys)
}

func (remote *S3Remote) ParseTag(repo, tag string) (ID, error) {
  bucket := remote.getBucket()

  file, err := bucket.Get(remote.tagFilePath(repo, tag))
  if s3err, ok := err.(*s3.Error); ok && s3err.StatusCode == 404 {
    // doesn't exist yet, deal with it
    return "", nil
  } else if err != nil {
    return "", err
  }

  return ID(file), nil
}

func (remote *S3Remote) ResolveImageNameToId(image string) (ID, error) {
  return ResolveImageNameToId(remote, image)
}

func (remote *S3Remote) ImageFullId(id ID) (ID, error) {
  remoteKeys, err := remote.repoKeys("/images")
  if err != nil {
    return "", err
  }

  for key, _ := range remoteKeys {
    parts := strings.Split(key, "/")
    if strings.HasPrefix(string(id), parts[0]) {
      return ID(parts[0]), nil
    }
  }

  return "", ErrNoSuchImage
}

func (remote *S3Remote) WalkImages(id ID, walker ImageWalkFn) error {
  return WalkImages(remote, id, walker)
}

func (remote *S3Remote) ImageMetadata(id ID) (client.Image, error) {
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
    key.Key = strings.TrimPrefix(name, remote.KeyPrefix + "/")
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

  fmt.Println("hello")
  return putFileMulti(remote.getBucket(), key, f, finfo.Size(), S3MinPartSize, "application/octet-stream", s3.Private)

  // return remote.getBucket().PutReader(key, buff, finfo.Size(), "application/octet-stream", s3.Private)
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
  fullKey := path.Join(remote.KeyPrefix, key)

  from, err := remote.getBucket().GetReader(fullKey)
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

  wrote,err := io.Copy(to, bufFrom)
  if err != nil {
    return err
  }

  fmt.Printf("pulled key %s (%s)\n", key, utils.HumanSize(wrote))

  return nil
}

// path to a tagfile
func (remote *S3Remote) tagFilePath(repo, tag string) string {
  return filepath.Join(remote.KeyPrefix, "repositories", repo, tag)
}

// path to an image dir
func (remote *S3Remote) imagePath(id ID) string {
  return filepath.Join(remote.KeyPrefix, "images", string(id))
}
