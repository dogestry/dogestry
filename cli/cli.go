package cli

import (
	"encoding/json"
	"flag"
	"fmt"
	"github.com/dogestry/dogestry/config"
	"github.com/dogestry/dogestry/remote"
	"github.com/dogestry/dogestry/utils"
	docker "github.com/fsouza/go-dockerclient"
	"io"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"reflect"
	"strings"
)

func NewDogestryCli(cfg config.Config) (*DogestryCli, error) {
	dogestryCli := &DogestryCli{
		Config:     cfg,
		err:        os.Stderr,
		DockerHost: cfg.GetDockerHost(),
	}

	var err error
	var newClient *docker.Client
	dockerCertPath := os.Getenv("DOCKER_CERT_PATH")

	if dockerCertPath != "" {
		cert := path.Join(dockerCertPath, "cert.pem")
		key := path.Join(dockerCertPath, "key.pem")
		ca := path.Join(dockerCertPath, "ca.pem")

		newClient, err = docker.NewTLSClient(dogestryCli.DockerHost, cert, key, ca)
	} else {
		newClient, err = docker.NewClient(dogestryCli.DockerHost)
	}

	if err != nil {
		return nil, err
	}
	dogestryCli.Client = newClient

	fmt.Printf("Using docker endpoint: %v\n", dogestryCli.DockerHost)

	return dogestryCli, nil
}

type DogestryCli struct {
	Client      *docker.Client
	err         io.Writer
	TempDir     string
	TempDirRoot string
	DockerHost  string
	Config      config.Config
}

func (cli *DogestryCli) getMethod(name string) (func(...string) error, bool) {
	methodName := "Cmd" + strings.ToUpper(name[:1]) + strings.ToLower(name[1:])
	method := reflect.ValueOf(cli).MethodByName(methodName)
	if !method.IsValid() {
		return nil, false
	}
	return method.Interface().(func(...string) error), true
}

func (cli *DogestryCli) RunCmd(args ...string) error {
	if len(args) > 0 {
		method, exists := cli.getMethod(args[0])
		if !exists {
			fmt.Println("Error: Command not found:", args[0])
			return cli.CmdHelp(args[1:]...)
		}
		return method(args[1:]...)
	}
	return cli.CmdHelp(args...)
}

func (cli *DogestryCli) CmdHelp(args ...string) error {
	if len(args) > 0 {
		method, exists := cli.getMethod(args[0])
		if !exists {
			fmt.Fprintf(cli.err, "Error: Command not found: %s\n", args[0])
		} else {
			method("--help")
			return nil
		}
	}

	help := fmt.Sprintf(
		`Usage: dogestry [OPTIONS] COMMAND [arg...]
Alternate registry and simple image storage for docker.
  Typical S3 Usage:
     export AWS_ACCESS_KEY=ABC
     export AWS_SECRET_KEY=DEF
     export DOCKER_HOST=tcp://localhost:2375
     dogestry push s3://<bucket name>/<path name>/?region=us-east-1 <image name>
     dogestry pull s3://<bucket name>/<path name>/?region=us-east-1 <image name>
     dogestry -tempdir /tmp download s3://<bucket name>/<path name>/?region=us-east-1 <image name>
     dogestry upload <image dir> <image name>
  Commands:
  	 download - Download IMAGE from S3 and save it locally to -tempdir. TAG defaults to 'latest'
  	 upload   - Upload image saved on IMAGE_DIR into docker
	 pull     - Pull IMAGE from S3 and load it into docker. TAG defaults to 'latest'
	 push     - Push IMAGE to S3. TAG defaults to 'latest'
	 remote   - Check a remote
`)
	fmt.Println(help)
	return nil
}

func (cli *DogestryCli) Subcmd(name, signature, description string) *flag.FlagSet {
	flags := flag.NewFlagSet(name, flag.ContinueOnError)
	flags.Usage = func() {
		fmt.Fprintf(cli.err, "\nUsage: dogestry %s %s\n\n%s\n\n", name, signature, description)
		flags.PrintDefaults()
		os.Exit(2)
	}
	return flags
}

// CreateAndReturnTempDir creates and returns temporary work dir
// This dir is cleaned up on exit
func (cli *DogestryCli) CreateAndReturnTempDir() string {
	if cli.TempDir == "" {
		if cli.TempDirRoot != "" {
			if err := os.MkdirAll(cli.TempDirRoot, 0755); err != nil {
				log.Fatal(err)
			}
			cli.TempDir = cli.TempDirRoot

		} else {
			if tempDir, err := ioutil.TempDir(cli.TempDirRoot, "dogestry"); err != nil {
				log.Fatal(err)
			} else {
				cli.TempDir = tempDir
			}
		}
	}

	return cli.TempDir
}

// WorkDirGivenBaseDir creates temporary dir
func (cli *DogestryCli) WorkDirGivenBaseDir(basedir, suffix string) (string, error) {
	suffix = strings.Replace(suffix, ":", "_", -1)

	path := filepath.Join(basedir, suffix)

	fmt.Printf("WorkDir: %v\n", path)

	if err := os.MkdirAll(path, os.ModeDir|0700); err != nil {
		return "", err
	}

	return path, nil
}

// WorkDir creates temporary dir
func (cli *DogestryCli) WorkDir(suffix string) (string, error) {
	suffix = strings.Replace(suffix, ":", "_", -1)
	basedir := cli.CreateAndReturnTempDir()

	return cli.WorkDirGivenBaseDir(basedir, suffix)
}

// clean up the tempDir
func (cli *DogestryCli) Cleanup() {
	if cli.TempDir != "" {
		if err := os.RemoveAll(cli.TempDir); err != nil {
			log.Println(err)
		}
	}
}

func (cli *DogestryCli) getLayerIdsToDownload(fromId remote.ID, imageRoot string, r remote.Remote) ([]remote.ID, error) {
	toDownload := make([]remote.ID, 0)

	err := r.WalkImages(fromId, func(id remote.ID, image docker.Image, err error) error {
		fmt.Printf("Examining id '%s' on remote docker host...\n", id.Short())
		if err != nil {
			return err
		}

		_, err = cli.Client.InspectImage(string(id))

		if err == docker.ErrNoSuchImage {
			toDownload = append(toDownload, id)
			return nil
		} else if err != nil {
			return err
		} else {
			fmt.Printf("Docker host already has id '%s', stop scanning.\n", id.Short())
			return remote.BreakWalk
		}

		return nil
	})

	return toDownload, err
}

func (cli *DogestryCli) pullImage(fromId remote.ID, imageRoot string, r remote.Remote) error {
	toDownload, err := cli.getLayerIdsToDownload(fromId, imageRoot, r)
	if err != nil {
		return err
	}

	for _, id := range toDownload {
		downloadPath := filepath.Join(imageRoot, string(id))

		fmt.Printf("Pulling image id '%s' to: %v\n", id.Short(), downloadPath)

		err := r.PullImageId(id, downloadPath)
		if err != nil {
			return err
		}
	}

	return nil
}

func (cli *DogestryCli) createRepositoriesJsonFile(image, imageRoot string, r remote.Remote) error {
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

// sendTar streams exported tarball into remote docker
func (cli *DogestryCli) sendTar(imageRoot string) error {
	notExist, err := utils.DirNotExistOrEmpty(imageRoot)

	if err != nil {
		return err
	}
	if notExist {
		fmt.Println("local directory is empty")
		return nil
	}

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

	cli.Client.LoadImage(docker.LoadImageOptions{InputStream: stdout})

	return nil
}
