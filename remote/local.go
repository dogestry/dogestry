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

type LocalRemote struct {
	Url      url.URL
	Path     string
}

func NewLocalRemote(url url.URL) (*LocalRemote, error) {
	// TODO validate

	return &LocalRemote{
		Url:      url,
		Path:     url.Path,
	}, nil
}

func (remote *LocalRemote) Desc() string {
	return fmt.Sprintf("local(%s)", remote.Path)
}

func (remote *LocalRemote) Push(image, imageRoot string) error {
	log.Println("pushing local", remote.Url.Path)

	return remote.rsyncTo(imageRoot, "")
}

// pull image into imageRoot
func (remote *LocalRemote) PullImageId(id, imageRoot string) error {
	log.Println("pushing local", remote.Url.Path)

	return remote.rsyncFrom("images/"+id, id)
}

// TODO make this truly remote
func (remote *LocalRemote) ResolveImageNameToId(image string) (string, error) {
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

func (remote *LocalRemote) WalkImages(id string, walker ImageWalkFn) error {
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
func (remote *LocalRemote) ParseTag(repo, tag string) (string, error) {
	repoPath := filepath.Join(filepath.Clean(remote.Url.Path), "repositories", repo, tag)

	if id, err := ioutil.ReadFile(repoPath); err == nil {
		return string(id), nil
	} else if os.IsNotExist(err) {
		return "", nil
	} else {
		return "", err
	}
}

func (remote *LocalRemote) ImageMetadata(id string) (client.Image, error) {
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
func (remote *LocalRemote) rsyncTo(src, dst string) error {
	return remote.rsync(src+"/", remote.RemotePath(dst)+"/")
}

func (remote *LocalRemote) rsyncFrom(src, dst string) error {
	return remote.rsync(remote.RemotePath(src)+"/", dst+"/")
}

func (remote *LocalRemote) rsync(src, dst string) error {
	fmt.Println("rsync", "-av", src, dst)
	out, err := exec.Command("rsync", "-av", src, dst).CombinedOutput()
	if err != nil {
		return fmt.Errorf("rsync failed: %s\noutput: %s", err, string(out))
	}
	log.Println(string(out))

	return nil
}

func (remote *LocalRemote) imagePath(id string) string {
	return remote.RemotePath("images", id)
}

func (remote *LocalRemote) RemotePath(part ...string) string {
	return filepath.Join(remote.Path, filepath.Join(part...))
}
