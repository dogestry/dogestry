package remote

import (
	"fmt"
	"log"
	"net/url"
	"os/exec"
	"path/filepath"
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
