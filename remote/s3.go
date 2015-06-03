package remote

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"

	"github.com/crowdmob/goamz/aws"
	"github.com/crowdmob/goamz/s3"
	"github.com/dogestry/dogestry/utils"
	docker "github.com/fsouza/go-dockerclient"
	"github.com/rlmcpherson/s3gof3r"
)

func NewS3Remote(config RemoteConfig) (*S3Remote, error) {
	s3, err := newS3Client(config)
	if err != nil {
		return &S3Remote{}, err
	}

	s3gof3rKeys := s3gof3r.Keys{AccessKey: config.Config.S3.Access_Key_Id, SecretKey: config.Config.S3.Secret_Key}

	return &S3Remote{
		config:               config,
		BucketName:           config.Url.Host,
		client:               s3,
		uploadDownloadClient: s3gof3r.New("", s3gof3rKeys),
	}, nil
}

type S3Remote struct {
	config               RemoteConfig
	BucketName           string
	Bucket               *s3.Bucket
	client               *s3.S3
	uploadDownloadClient *s3gof3r.S3
}

var (
	S3DefaultRegion = "us-east-1"
)

// create a new s3 client from the url
func newS3Client(config RemoteConfig) (*s3.S3, error) {
	auth, err := aws.GetAuth(config.Config.S3.Access_Key_Id, config.Config.S3.Secret_Key, "", time.Now())
	if err != nil {
		return &s3.S3{}, err
	}

	var regionName string

	regQuery := config.Url.Query()["region"]

	if len(regQuery) > 0 && regQuery[0] != "" {
		regionName = regQuery[0]
	} else {
		regionName = S3DefaultRegion
	}

	region := aws.Regions[regionName]

	return s3.New(auth, region), nil
}

func (remote *S3Remote) Validate() error {
	bucket := remote.getBucket()

	_, err := bucket.List("", "", "", 1)
	if err != nil {
		return fmt.Errorf("%s unable to ping s3 bucket: %s", remote.Desc(), err)
	}

	return nil
}

// Remote: describe the remote
func (remote *S3Remote) Desc() string {
	return fmt.Sprintf("s3(bucket=%s, region=%s)", remote.BucketName, remote.client.Region.Name)
}

type putFileTuple struct {
	Key    string
	KeyDef keyDef
}

func makeFilesChan(keysToPush keys) <-chan putFileTuple {
	putFilesChan := make(chan putFileTuple, len(keysToPush))
	go func() {
		defer close(putFilesChan)
		for key, localKey := range keysToPush {
			keyDefClone := *localKey
			putFilesChan <- putFileTuple{key, keyDefClone}
		}
	}()
	return putFilesChan
}

func (remote *S3Remote) Push(image, imageRoot string) error {
	var err error

	keysToPush, err := remote.localKeys(imageRoot)
	if err != nil {
		return fmt.Errorf("error calculating keys to push: %v", err)
	}

	if len(keysToPush) == 0 {
		log.Println("There are no files to push")
		return nil
	}

	type putFileResult struct {
		host string
		err  error
	}

	putFileErrChan := make(chan putFileResult)
	putFilesChan := makeFilesChan(keysToPush)

	defer close(putFileErrChan)

	numGoroutines := 100

	log.Println("Pushing files to S3 remote")
	for i := 0; i < numGoroutines; i++ {
		go func() {
			for putFile := range putFilesChan {
				putFileErr := remote.putFile(putFile.KeyDef.fullPath, &putFile.KeyDef)

				if (putFileErr != nil) && ((putFileErr != io.EOF) && (!strings.Contains(putFileErr.Error(), "EOF"))) {
					putFileErrChan <- putFileResult{putFile.Key, putFileErr}
					return
				}
			}
		}()
	}

	for i := 0; i < len(keysToPush); i++ {
		p := <-putFileErrChan
		if p.err != nil {
			log.Printf("error when uploading to S3: %v", p.err)
			return fmt.Errorf("Error when uploading to S3: %v", p.err)

		}
	}

	return err
}

func (remote *S3Remote) PullImageId(id ID, dst string) error {
	rootKey := "images/" + string(id)
	imageKeys, err := remote.repoKeys("/" + rootKey)
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

func (remote *S3Remote) ImageMetadata(id ID) (docker.Image, error) {
	jsonPath := path.Join(remote.imagePath(id), "json")
	image := docker.Image{}

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

func (remote *S3Remote) ParseImagePath(path string, prefix string) (repo, tag string) {
	return ParseImagePath(path, prefix)
}

func (remote *S3Remote) getBucket() *s3.Bucket {
	return remote.client.Bucket(remote.BucketName)
}

func (remote *S3Remote) getUploadDownloadBucket() *s3gof3r.Bucket {
	return remote.uploadDownloadClient.Bucket(remote.config.Url.Host)
}

type keyDef struct {
	key    string
	sumKey string

	sum string

	s3Key    s3.Key
	fullPath string

	remote *S3Remote
}

// keys represents either local or remote files
type keys map[string]*keyDef

// gets a key, creating the underlying keyDef if required
// we need to S3Remote for getting the sum, so add it here
func (k keys) Get(key string, remote *S3Remote) *keyDef {
	if existing, ok := k[key]; ok {
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

	for key, thisKeyDef := range k {
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
	bytesSum, err := kd.remote.getBucket().Get(kd.sumKey)
	if err != nil {
		return ""
	}

	kd.sum = string(bytesSum)

	return kd.sum
}

// get repository keys from s3
func (remote *S3Remote) repoKeys(prefix string) (keys, error) {
	repoKeys := make(keys)

	prefix = strings.Trim(prefix, "/")

	bucket := remote.getBucket()

	cnt, err := bucket.List(prefix, "", "", 1000)

	if err != nil {
		return repoKeys, fmt.Errorf("getting bucket contents at prefix '%s': %s", prefix, err)
	}

	for _, key := range cnt.Contents {
		if key.Key == "" {
			continue
		}

		plainKey := strings.TrimPrefix(key.Key, "/")

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
			key:      key,
			sum:      sum,
			fullPath: path,
		}

		return nil
	})

	if err != nil {
		return localKeys, err
	}

	return localKeys, nil
}

type progress struct {
	worker    int
	size      int64
	totalSize int64
	index     int
	err       error
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

	progressReader := utils.NewProgressReader(f, finfo.Size(), src)

	// Open a PutWriter for actual file upload
	w, err := remote.getUploadDownloadBucket().PutWriter(f.Name(), nil, nil)
	if err != nil {
		return err
	}
	if _, err = io.Copy(w, progressReader); err != nil { // Copy to S3
		return err
	}
	if err = w.Close(); err != nil {
		return err
	}

	// Open a second PutWriter for checksum upload.
	w2, err := remote.getUploadDownloadBucket().PutWriter(dstKey+".sum", nil, nil)
	if err != nil {
		return err
	}
	if _, err = io.Copy(w2, strings.NewReader(key.Sum())); err != nil { // Copy to S3
		return err
	}
	if err = w2.Close(); err != nil {
		return err
	}

	return nil
}

// get files from the s3 bucket to a local path, relative to rootKey
// eg
//
// dst: "/tmp/rego/123"
// rootKey: "images/456"
// key: "images/456/json"
// downloads to: "/tmp/rego/123/456/json"
func (remote *S3Remote) getFiles(dst, rootKey string, imageKeys keys) error {
	getFilesErrMap := make(map[string]error)

	type errTuple struct {
		fileKey string
		err     error
	}

	tupleCh := make(chan errTuple)

	for _, key := range imageKeys {
		keyDefClone := *key

		go func(dst, rootKey string, key keyDef) {
			relKey := strings.TrimPrefix(key.key, rootKey)
			relKey = strings.TrimPrefix(relKey, "/")

			err := remote.getFile(filepath.Join(dst, relKey), &key)
			if err != nil {
				tupleCh <- errTuple{key.key, err}
				return
			}
			tupleCh <- errTuple{"", nil}

		}(dst, rootKey, keyDefClone)
	}

	for range imageKeys {
		tuple := <-tupleCh
		if tuple.err != nil {
			getFilesErrMap[tuple.fileKey] = tuple.err
		}
	}
	close(tupleCh)

	if len(getFilesErrMap) > 0 {
		log.Printf("Errors during getFiles: %v", getFilesErrMap)
		return fmt.Errorf("error downloading files from S3")
	}

	return nil
}

// get a single file from the s3 bucket
func (remote *S3Remote) getFile(dst string, key *keyDef) error {
	log.Printf("Pulling key %s (%s)\n", key.key, utils.HumanSize(key.s3Key.Size))

	from, _, err := remote.getUploadDownloadBucket().GetReader(key.key, nil)
	if err != nil {
		return err
	}
	defer from.Close()

	if err := os.MkdirAll(filepath.Dir(dst), 0700); err != nil {
		return err
	}

	to, err := os.Create(dst)
	if err != nil {
		return err
	}

	progressReaderFrom := utils.NewProgressReader(from, key.s3Key.Size, key.key)

	_, err = io.Copy(to, progressReaderFrom)
	if err != nil {
		return err
	}

	return nil
}

// path to a tagfile
func (remote *S3Remote) tagFilePath(repo, tag string) string {
	return filepath.Join("repositories", repo, tag)
}

// path to an image dir
func (remote *S3Remote) imagePath(id ID) string {
	return filepath.Join("images", string(id))
}

func (remote *S3Remote) remoteKey(key string) string {
	return key
}

func (remote *S3Remote) List() (images []Image, err error) {

	bucket := remote.getBucket()
	nextMarker := ""

	var contents []s3.Key

	for true {
		resp, err := bucket.List("repositories/", "", nextMarker, 1000)
		if err != nil {
			log.Printf("%s unable to list images: %s", remote.Desc(), err)
			return images, err
		}

		contents = append(contents, resp.Contents...)

		if resp.IsTruncated {
			nextMarker = resp.NextMarker
		} else {
			break
		}
	}

	for _, k := range contents {
		if strings.HasSuffix(k.Key, ".sum") {
			continue
		}
		repo, tag := remote.ParseImagePath(k.Key, "repositories/")
		if err != nil {
			log.Printf("error splitting S3 key: repositories/")
			return images, err
		}

		image := Image{repo, tag}
		images = append(images, image)
	}

	return images, nil
}
