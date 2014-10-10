package cli

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"

	docker "github.com/fsouza/go-dockerclient"
	"github.com/newrelic-forks/dogestry/remote"
)

func (cli *DogestryCli) CmdPull(args ...string) error {
	cmd := cli.Subcmd("pull", "REMOTE IMAGE[:TAG]", "pull IMAGE from the REMOTE and load it into docker. TAG defaults to 'latest'")
	if err := cmd.Parse(args); err != nil {
		return nil
	}

	if len(cmd.Args()) < 2 {
		return fmt.Errorf("Error: REMOTE and IMAGE not specified")
	}

	remoteDef := cmd.Arg(0)
	image := cmd.Arg(1)

	imageRoot, err := cli.WorkDir(image)
	if err != nil {
		return err
	}
	r, err := remote.NewRemote(remoteDef, cli.Config)
	if err != nil {
		return err
	}

	fmt.Println("remote", r.Desc())

	fmt.Printf("resolving image id for: %v\n", image)
	id, err := r.ResolveImageNameToId(image)
	if err != nil {
		return err
	}

	fmt.Printf("image '%s' resolved on remote id '%s'\n", image, id.Short())

	fmt.Println("preparing images")
	if err := cli.preparePullImage(id, imageRoot, r); err != nil {
		return err
	}

	fmt.Println("preparing repositories file")
	if err := prepareRepositories(image, imageRoot, r); err != nil {
		return err
	}

	fmt.Println("sending tar to docker")
	if err := cli.sendTar(image, id.String(), imageRoot); err != nil {
		return err
	}

	return nil
}

func (cli *DogestryCli) preparePullImage(fromId remote.ID, imageRoot string, r remote.Remote) error {
	toDownload := make([]remote.ID, 0)

	err := r.WalkImages(fromId, func(id remote.ID, image docker.Image, err error) error {
		fmt.Printf("examining id '%s' on remote\n", id.Short())
		if err != nil {
			fmt.Println("err", err)
			return err
		}

		_, err = cli.client.InspectImage(string(id))
		if err == docker.ErrNoSuchImage {
			toDownload = append(toDownload, id)
			return nil
		} else if err != nil {
			return err
		} else {
			fmt.Printf("docker already has id '%s', stopping\n", id.Short())
			return remote.BreakWalk
		}
	})

	if err != nil {
		return err
	}

	for _, id := range toDownload {
		if err := cli.pullImage(id, filepath.Join(imageRoot, string(id)), r); err != nil {
			return err
		}
	}

	return nil
}

func (cli *DogestryCli) pullImage(id remote.ID, dst string, r remote.Remote) error {
	fmt.Printf("Pulling image id '%s' to dst: %v\n", id.Short(), dst)

	return r.PullImageId(id, dst)
}

func prepareRepositories(image, imageRoot string, r remote.Remote) error {
	repoName, repoTag := remote.NormaliseImageName(image)

	id, err := r.ParseTag(repoName, repoTag)
	if err != nil {
		return err
	} else if id == "" {
		return nil
	}

	reposPath := filepath.Join(imageRoot, "repositories")
	reposFile, err := os.Create(reposPath)
	if err != nil {
		return err
	}
	defer reposFile.Close()

	repositories := map[string]Repository{}
	repositories[repoName] = Repository{}
	repositories[repoName][repoTag] = string(id)

	return json.NewEncoder(reposFile).Encode(&repositories)
}

// stream the tarball into docker
// its easier here to use tar command, but it'd be neater to mirror Push's approach
func (cli *DogestryCli) sendTar(id, tag, imageRoot string) error {
	notExist, err := dirNotExistOrEmpty(imageRoot)

	if err != nil {
		return err
	}
	if notExist {
		fmt.Println("no images to send to docker")
		return nil
	}

	// DEBUG - write out a tar to see what's there!
	// exec.Command("/bin/tar", "cvf", "/tmp/d.tar", "-C", imageRoot, ".").Run()

	cmd := exec.Command("tar", "cvf", "-", "-C", imageRoot, ".")
	cmd.Env = os.Environ()
	cmd.Dir = imageRoot
	defer cmd.Wait()

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return err
	}

	if err := cmd.Start(); err != nil {
		return err
	}

	fmt.Println("kicking off post")

	return cli.client.LoadImage(docker.LoadImageOptions{InputStream: stdout})
}

func dirNotExistOrEmpty(path string) (bool, error) {
	imagesDir, err := os.Open(path)
	if err != nil {
		// no images
		if os.IsNotExist(err) {
			return true, nil
		} else {
			return false, err
		}
	}
	defer imagesDir.Close()

	names, err := ioutil.ReadDir(path)
	if err != nil {
		return false, err
	}

	if len(names) <= 1 {
		return true, nil
	}

	return false, nil
}
