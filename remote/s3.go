package remote

import (
  "github.com/lachie/goamz/aws"
  "github.com/lachie/goamz/s3"
  "dogestry/utils"

  "bufio"
  "dogestry/client"
	"dogestry/compressor"
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
  compressor compressor.Compressor
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

  compressor,err := compressor.NewCompressor(config.Config)
  if err != nil {
    return nil,err
  }

  return &S3Remote{
    config: config,
    BucketName: url.Host,
    KeyPrefix:  prefix,
    client:     s3,
    compressor: compressor,
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
  fmt.Println("fetching repo keys")
  remoteKeys, err := remote.repoKeys("")
  if err != nil {
    return fmt.Errorf("error getting repoKeys: %s", err)
  }

  fmt.Println("fetching local keys")
  localKeys, err := remote.localKeys(imageRoot)
  if err != nil {
    return fmt.Errorf("error getting localKeys: %s", err)
  }


  // DEBUG
  //delete(remoteKeys, "images/8dbd9e392a964056420e5d58ca5cc376ef18e2de93b5cc90e868a1bbc8318c1c/layer.tar.lz4")


  fmt.Println("comparing keys")
  keysToPush := localKeys.NotIn(remoteKeys)

  if len(keysToPush) == 0 {
    fmt.Println("nothing to push")
    return nil
  }

  for key, localKey := range keysToPush {
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



type keyDef struct {
  key string
  sumKey string

  sum string

  s3Key s3.Key
  fullPath string

  remote *S3Remote
}


// keys represents either local or remote files
type keys map[string]*keyDef

// gets a key, creating the underlying keyDef if required
// we need to S3Remote for getting the sum, so add it here
func (k keys) Get(key string, remote *S3Remote) *keyDef {
  if existing,ok := k[key]; ok {
    return existing
  } else {
    k[key] = &keyDef{key: key, remote: remote}
  }

  return k[key]
}




// Returns keys either not existing in other, 
// or whose sum doesn't match.
func (k keys) NotIn(other keys) keys {
  notIn := make(keys)

  for key,thisKeyDef := range k {
    if otherKeyDef, ok := other[key]; !ok || otherKeyDef.Sum() != thisKeyDef.Sum() {
      notIn[key] = thisKeyDef
    }
  }

  return notIn
}


func (kd *keyDef) Sum() (sum string) {
  if kd.sum != "" {
    return kd.sum
  }

  if kd.sumKey == "" {
    return ""
  }

  // get sum!
  // honestly there's not much we can do if we don't get the sum here
  // maybe a panic??
  bytesSum,err := kd.remote.getBucket().Get(kd.sumKey)
  if err != nil {
    return ""
  }

  kd.sum = string(bytesSum)

  return kd.sum
}




// get repository keys from s3
func (remote *S3Remote) repoKeys(prefix string) (keys, error) {
  repoKeys := make(keys)
  remotePrefix := remote.KeyPrefix + "/"
  bucketPrefix := remote.KeyPrefix + prefix

  bucket := remote.getBucket()

  cnt, err := bucket.GetBucketContentsWithPrefix(bucketPrefix)
  if err != nil {
    return repoKeys, fmt.Errorf("getting bucket contents at prefix '%s': %s", prefix, err)
  }

  for _, key := range *cnt {
    if key.Key == "" {
      continue
    }

    plainKey := strings.TrimPrefix(key.Key, remotePrefix)

    if strings.HasSuffix(plainKey, ".sum") {
      plainKey = strings.TrimSuffix(plainKey, ".sum")
      repoKeys.Get(plainKey, remote).sumKey = key.Key

    } else {
      repoKeys.Get(plainKey, remote).s3Key = key
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

    // note that we pre-populate the sum here
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

  progressReader := utils.NewProgressReader(f, finfo.Size(), os.Stdout)

  // XXX We don't know how big the file will be ahead of time!
  //compressorReader,err := remote.compressor.CompressReader(progressReader)
  //if err != nil {
    //return err
  //}

  err = remote.getBucket().PutReader(dstKey, progressReader, finfo.Size(), "application/octet-stream", s3.Private)
  if err != nil {
    return err
  }

  return remote.getBucket().Put(dstKey+".sum", []byte(key.Sum()), "text/plain", s3.Private)
}




// get files from the s3 bucket to a local path, relative to rootKey
// eg
//
// dst: "/tmp/rego/123"
// rootKey: "images/456"
// key: "images/456/json"
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



// TODO readd
      //// special case - compress layer.tar
      //if filepath.Base(dest) == "layer.tar" {
        //if err := cli.compressor.Compress(dest); err != nil {
          //return err
        //}
      //}
