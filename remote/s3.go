package remote

import (
  "github.com/lachie/goamz/aws"
  "github.com/lachie/goamz/s3"
  "dogestry/utils"

  "bufio"
  "dogestry/client"
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


type keyDef struct {
  key string
  s3Key s3.Key
  sum string
  fullPath string
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
  //delete(remoteKeys, "images/8dbd9e392a964056420e5d58ca5cc376ef18e2de93b5cc90e868a1bbc8318c1c/layer.tar.lz4")

  for key, localKey := range localKeys.NotIn(remoteKeys) {
    fmt.Printf("pushing key %s (%s)\n", key, utils.FileHumanSize(localKey.fullPath))

    if err := remote.putFile(localKey.fullPath, localKey); err != nil {
      return err
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
    key = strings.TrimPrefix(key, "images/")
    parts := strings.Split(key, "/")
    if strings.HasPrefix(parts[0], string(id)) {
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



// keys represents either local or remote files
type keys map[string]*keyDef

// gets a key, creating the underlying keyDef if required
func (k keys) Get(key string) *keyDef {
  if existing,ok := k[key]; ok {
    return existing
  } else {
    k[key] = &keyDef{key: key}
  }

  return k[key]
}

// Returns keys either not existing in other, 
// or whose sum doesn't match.
func (k keys) NotIn(other keys) keys {
  notIn := make(keys)

  for key,thisKeyDef := range k {
    if otherKeyDef, ok := other[key]; !ok || otherKeyDef.sum != thisKeyDef.sum {
      notIn[key] = thisKeyDef
    }
  }

  return notIn
}



// get repository keys from s3
func (remote *S3Remote) repoKeys(prefix string) (keys, error) {
  repoKeys := make(keys)
  remotePrefix := remote.KeyPrefix + "/"

  bucket := remote.getBucket()

  cnt, err := bucket.GetBucketContentsWithPrefix(remote.KeyPrefix + prefix)
  if err != nil {
    return repoKeys, err
  }

  for _, key := range *cnt {
    if key.Key == "" {
      continue
    }

    plainKey := strings.TrimPrefix(key.Key, remotePrefix)

    if strings.HasSuffix(plainKey, ".sum") {
      plainKey = strings.TrimSuffix(plainKey, ".sum")

      bytesSum,err := bucket.Get(key.Key)
      if err != nil {
        return repoKeys, err
      }

      repoKeys.Get(plainKey).sum = string(bytesSum)

    } else {
      repoKeys.Get(plainKey).s3Key = key
    }
  }

  return repoKeys, nil
}

// Get repository keys from the local work dir.
// Returned as a map of s3.Key's for ease of comparison.
func (remote *S3Remote) localKeys(root string) (keys, error) {
  localKeys := make(keys)

  if root[len(root)-1] != '/' {
    root = root + "/"
  }

  err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
    if info.IsDir() {
      return nil
    }

    sum, err := utils.Sha1File(path)
    if err != nil {
      return err
    }

    key := strings.TrimPrefix(path, root)

    localKeys[key] = &keyDef{
      key: key,
      sum: sum,
      fullPath: path,
    }

    return nil
  })

  // XXX hmmm
  if err != nil {
    return localKeys, nil
  }

  return localKeys, nil
}


type progress struct {
  worker int
  size int64
  totalSize int64
  index int
  err error
}

// put a file with key from imageRoot to the s3 bucket
func (remote *S3Remote) putFile(src string, key *keyDef) error {
  dstKey := remote.remoteKey(key.key)

  f, err := os.Open(src)
  if err != nil {
    return err
  }
  defer f.Close()

  finfo, err := f.Stat()
  if err != nil {
    return err
  }



  // TODO make these two operations more atomic somehow
  p,err := remote.getBucket().NewParallelUploaderFromReaderAt(dstKey, f, finfo.Size())
  if err != nil {
    return err
  }

  p.WorkerCount = 4

  progressc := make(chan progress, 100)

  go func() {
    for p := range progressc {
      fmt.Println("progress", p.size, p.index)
    }
  }()
  defer close(progressc)

  p.UploadWorker = func (id int, job s3.PartJob) error {
    fmt.Println("starting", job.Part.N)
    err := s3.MultiPartUploader(id, job)

    fmt.Printf("got %#v", job)

    progressc <- progress{
      size: job.Size,
      worker: id,
      totalSize: p.TotalSize,
      index: job.Part.N,
      err: err,
    }

    return err
  }

  fmt.Println("putting")
  if err := p.Put(); err != nil {
    return err
  }

  return remote.getBucket().Put(dstKey+".sum", []byte(key.sum), "text/plain", s3.Private)
}




// get files from the s3 bucket to a local path, relative to rootKey
// eg
//
// dst: "/tmp/rego/123"
// rootKey: "images/456"
// key: "images/images/456/json"
// downloads to: "/tmp/rego/123/456/json"
func (remote *S3Remote) getFiles(dst, rootKey string, imageKeys keys) error {
  for _, keyDef := range imageKeys {
    relKey := strings.TrimPrefix(keyDef.key, rootKey)
    err := remote.getFile(filepath.Join(dst, relKey), keyDef)
    if err != nil {
      return err
    }
  }

  return nil
}

// get a single file from the s3 bucket
func (remote *S3Remote) getFile(dst string, key *keyDef) error {
  srcKey := remote.remoteKey(key.key)

  from, err := remote.getBucket().GetReader(srcKey)
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

  // TODO validate against sum

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

// the full remote key (adds KeyPrefix)
func (remote *S3Remote) remoteKey(key string) string {
  return path.Join(remote.KeyPrefix, key)
}

