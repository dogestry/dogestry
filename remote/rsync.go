package remote

import (
	"dogestry/client"
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



type RsyncRemote struct {
	Url url.URL
}

func NewRsyncRemote(url url.URL) (*RsyncRemote, error) {
	// TODO validate

	return &RsyncRemote{
		Url: url,
	}, nil
}

func (remote *RsyncRemote) Desc() string {
	return remote.Url.String()
}

// 
func (remote *RsyncRemote) Push(image, imageRoot string) error {
	log.Println("pushing rsync", remote.Url.Path)

	src := filepath.Clean(imageRoot) + "/"
	dst := filepath.Clean(remote.Url.Path) + "/"

	log.Println("rsync", "-av", src, dst)
	out, err := exec.Command("rsync", "-av", src, dst).CombinedOutput()
	if err != nil {
		return fmt.Errorf("rsync failed: %s\noutput: %s", err, string(out))
	}
	log.Println(string(out))

	return nil
}

// pull image into imageRoot
func (remote *RsyncRemote) PullImageId(id, imageRoot string) error {
  log.Println("pushing rsync", remote.Url.Path)

	src := remote.imagePath(id) + "/"
	dst := filepath.Join(filepath.Clean(imageRoot), id) + "/"

	log.Println("rsync", "-av", src, dst)
	out, err := exec.Command("rsync", "-av", src, dst).CombinedOutput()
	if err != nil {
		return fmt.Errorf("rsync failed: %s\noutput: %s", err, string(out))
	}
	log.Println(string(out))

	return nil
}

func (remote *RsyncRemote) ResolveImageNameToId(image string) (string, error) {
	fmt.Println("hi resolving")

	// first, try the repos
	repoName, repoTag := NormaliseImageName(image)
	if id, err := remote.ParseTag(repoName, repoTag); err != nil {
		return "", err
	} else if id != "" {
		return id, nil
	}

	// ok, no repo
	//
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
		if strings.HasPrefix(name, image) {
			return name, nil
		}
	}

	return "", fmt.Errorf("no image '%s' found on %s", image, remote.Desc())
}



func (remote *RsyncRemote) WalkImages(id string, walker ImageWalkFn) error {
  if id == "" {
    return nil
  }

	img, err := remote.ImageMetadata(id)
	// image wasn't found
	if err != nil {
    return walker(id, client.Image{}, err)
	}

  err = walker(id, img, nil)
	if err != nil {
    // abort the walk
    if err == BreakWalk {
      return nil
    }
		return err
	}

	return remote.WalkImages(img.Parent, walker)
}

func (remote *RsyncRemote) ParseTag(repo, tag string) (string, error) {
	repoPath := filepath.Join(filepath.Clean(remote.Url.Path), "repositories", repo, tag)

	if id, err := ioutil.ReadFile(repoPath); err == nil {
		return string(id), nil
	} else if os.IsNotExist(err) {
		return "", nil
	} else {
		return "", err
	}
}

func (remote *RsyncRemote) ImageMetadata(id string) (client.Image, error) {
	image := client.Image{}

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


func rsync(src, dst string) error {
  return nil
}


func (remote *RsyncRemote) imagePath(id string) string {
	return filepath.Join(remote.repoRoot(), "images", id)
}

func (remote *RsyncRemote) repoRoot() string {
	return filepath.Clean(remote.Url.Path)
}

