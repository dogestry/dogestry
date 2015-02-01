package remote

import (
	docker "github.com/fsouza/go-dockerclient"

	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

type LocalRemote struct {
	config RemoteConfig
	Url    url.URL
	Path   string
}

func NewLocalRemote(config RemoteConfig) (*LocalRemote, error) {
	// TODO validate

	return &LocalRemote{
		config: config,
		Url:    config.Url,
		Path:   config.Url.Path,
	}, nil
}

func (remote *LocalRemote) Validate() error {
	return nil
}

func (remote *LocalRemote) Desc() string {
	return fmt.Sprintf("local(%s)", remote.Path)
}

// push all of imageRoot to the remote
func (remote *LocalRemote) Push(image, imageRoot string) error {
	log.Println("pushing local", remote.Url.Path)

	return remote.rsyncTo(imageRoot, "")
}

// pull image with id into dst
func (remote *LocalRemote) PullImageId(id ID, dst string) error {
	log.Println("pulling local", "images/"+id, "->", dst)

	return remote.rsyncFrom("images/"+string(id), dst)
}

func (remote *LocalRemote) ImageFullId(id ID) (ID, error) {
	// look for an image
	imagesRoot := filepath.Join(filepath.Clean(remote.Url.Path), "images")
	file, err := os.Open(imagesRoot)
	if err != nil {
		return "", err
	}

	names, err := file.Readdirnames(-1)
	if err != nil {
		return "", err
	}

	for _, name := range names {
		if strings.HasPrefix(name, string(id)) {
			return ID(name), nil
		}
	}

	return "", ErrNoSuchImage
}

func (remote *LocalRemote) WalkImages(id ID, walker ImageWalkFn) error {
	return WalkImages(remote, id, walker)
}

func (remote *LocalRemote) ResolveImageNameToId(image string) (ID, error) {
	return ResolveImageNameToId(remote, image)
}

func (remote *LocalRemote) ParseTag(repo, tag string) (ID, error) {
	repoPath := filepath.Join(filepath.Clean(remote.Url.Path), "repositories", repo, tag)

	if id, err := ioutil.ReadFile(repoPath); err == nil {
		return ID(id), nil
	} else if os.IsNotExist(err) {
		return "", nil
	} else {
		return "", err
	}
}

func (remote *LocalRemote) ImageMetadata(id ID) (docker.Image, error) {
	image := docker.Image{}

	imageJson, err := ioutil.ReadFile(filepath.Join(remote.imagePath(id), "json"))
	if os.IsNotExist(err) {
		return image, ErrNoSuchImage
	} else if err != nil {
		return image, err
	}

	if err := json.Unmarshal(imageJson, &image); err != nil {
		return image, err
	}

	return image, nil
}

func (remote *LocalRemote) ParseImagePath(path string, prefix string) (repo, tag string) {
	return ParseImagePath(path, prefix)
}

func (remote *LocalRemote) rsyncTo(src, dst string) error {
	return remote.rsync(src+"/", remote.RemotePath(dst)+"/")
}

func (remote *LocalRemote) rsyncFrom(src, dst string) error {
	return remote.rsync(remote.RemotePath(src)+"/", dst+"/")
}

func (remote *LocalRemote) rsync(src, dst string) error {
	out, err := exec.Command("rsync", "-av", src, dst).CombinedOutput()
	if err != nil {
		return fmt.Errorf("rsync failed: %s\noutput: %s", err, string(out))
	}
	log.Println(string(out))

	return nil
}

func (remote *LocalRemote) imagePath(id ID) string {
	return remote.RemotePath("images", string(id))
}

func (remote *LocalRemote) RemotePath(part ...string) string {
	return filepath.Join(remote.Path, filepath.Join(part...))
}

func (remote *LocalRemote) List() (images []Image, err error) {
	imagesRoot := filepath.Join(filepath.Clean(remote.Url.Path), "repositories")
	_, err = os.Open(imagesRoot)
	if err != nil {
		return images, err
	}

	pathList := []string{}
	err = filepath.Walk(imagesRoot, func(path string, info os.FileInfo, _ error) error {
		if !info.IsDir() {
			pathList = append(pathList, path)
		}
		return nil
	})
	if err != nil {
		return images, err
	}

	for _, path := range pathList {
		repo, tag := remote.ParseImagePath(path, imagesRoot+"/")
		if err != nil {
			fmt.Fprintf(os.Stderr, "error splitting repository key\n")
			return images, err
		}
		image := Image{repo, tag}
		images = append(images, image)
	}

	return images, nil
}
