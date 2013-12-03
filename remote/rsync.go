package remote

import (
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
	Url       url.URL
	Image     string
	ImageRoot string
}

func NewRsyncRemote(spec RemoteSpec) (*RsyncRemote, error) {
	// TODO validate

	return &RsyncRemote{
		Url:       spec.url,
		Image:     spec.image,
		ImageRoot: spec.imageRoot,
	}, nil
}

func (remote *RsyncRemote) Desc() string {
  return remote.Url.String()
}

func (remote *RsyncRemote) Push() error {
	log.Println("pushing rsync", remote.Url.Path)

	src := filepath.Clean(remote.ImageRoot) + "/"
	dst := filepath.Clean(remote.Url.Path) + "/"

	log.Println("rsync", "-av", src, dst)
	out, err := exec.Command("rsync", "-av", src, dst).CombinedOutput()
	if err != nil {
		return fmt.Errorf("rsync failed: %s\noutput: %s", err, string(out))
	}
	log.Println(string(out))

	return nil
}

func (remote *RsyncRemote) ResolveImageName(image string) (string, error) {
	fmt.Println("hi resolving")

  // first, try the repos
	repoName, repoTag := normaliseImageName(image)
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

func normaliseImageName(image string) (string, string) {
	repoParts := strings.Split(image, ":")
	if len(repoParts) == 1 {
		return repoParts[0], "latest"
	} else {
		return repoParts[0], repoParts[1]
	}
}
