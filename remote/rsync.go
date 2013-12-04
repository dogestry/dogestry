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
	Url      url.URL
	Hostname string
	Path     string
}

func NewRsyncRemote(url url.URL) (*RsyncRemote, error) {
	// TODO validate

	return &RsyncRemote{
		Url:      url,
		Hostname: url.Host,
		Path:     url.Path,
	}, nil
}

func (remote *RsyncRemote) Desc() string {
	return fmt.Sprintf("rsync(%s:%s)", remote.Hostname, remote.Path)
}

func (remote *RsyncRemote) Push(image, imageRoot string) error {
	log.Println("pushing rsync", remote.Url.Path)

	return remote.rsyncTo(imageRoot, "")
}

// pull image into imageRoot
func (remote *RsyncRemote) PullImageId(id, imageRoot string) error {
	log.Println("pushing rsync", remote.Url.Path)

	return remote.rsyncFrom("images/"+id, id)
}

// TODO make this truly remote
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

// TODO get this to work remotely (scp?)
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

	// TODO make this truly remote
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

// TODO factor this out
func (remote *RsyncRemote) rsyncTo(src, dst string) error {
	return rsync(src+"/", remote.RemotePath(dst)+"/")
}

func (remote *RsyncRemote) rsyncFrom(src, dst string) error {
	return rsync(remote.RemotePath(src)+"/", dst+"/")
}

func (remote *RsyncRemote) rsync(src, dst string) error {
	fmt.Println("rsync", "-av", src, dst)
	out, err := exec.Command("rsync", "-av", src, dst).CombinedOutput()
	if err != nil {
		return fmt.Errorf("rsync failed: %s\noutput: %s", err, string(out))
	}
	log.Println(string(out))

	return nil
}

func (remote *RsyncRemote) imagePath(id string) string {
	return remote.RemotePath("images", id)
}

func (remote *RsyncRemote) RemotePath(part ...string) string {
	path := filepath.Join(remote.Path, filepath.Join(part...))
	if remote.Hostname != "" {
		return remote.Hostname + ":" + path
	} else {
		return path
	}
}
