package cli

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"reflect"
	"strings"
	"time"

	"github.com/cheggaaa/pb"
	"github.com/dogestry/dogestry/config"
	"github.com/dogestry/dogestry/remote"
	docker "github.com/fsouza/go-dockerclient"
	homedir "github.com/mitchellh/go-homedir"
)

func newDockerClient(host string) (*docker.Client, error) {
	var err error
	var newClient *docker.Client
	dockerCertPath := os.Getenv("DOCKER_CERT_PATH")

	homeDir, _ := homedir.Dir()
	dockerConfigDir := path.Join(homeDir, ".docker")

	_, err = os.Stat(path.Join(dockerConfigDir, "cert.pem"))
	certExists := err == nil

	_, err = os.Stat(path.Join(dockerConfigDir, "ca.pem"))
	caExists := err == nil

	_, err = os.Stat(path.Join(dockerConfigDir, "key.pem"))
	keyExists := err == nil

	if dockerCertPath == "" && certExists && caExists && keyExists {
		dockerCertPath = dockerConfigDir
	}

	if dockerCertPath != "" {
		cert := path.Join(dockerCertPath, "cert.pem")
		key := path.Join(dockerCertPath, "key.pem")
		ca := path.Join(dockerCertPath, "ca.pem")

		newClient, err = docker.NewTLSClient(host, cert, key, ca)
	} else {
		newClient, err = docker.NewClient(host)
	}

	if err != nil {
		return nil, err
	}
	return newClient, err
}

func NewDogestryCli(cfg config.Config, hosts []string) (*DogestryCli, error) {
	var err error

	dogestryCli := &DogestryCli{
		Config:     cfg,
		err:        os.Stderr,
		DockerHost: cfg.Docker.Connection,
		PullHosts:  hosts,
	}

	dogestryCli.Client, err = newDockerClient(dogestryCli.DockerHost)
	if err != nil {
		log.Fatal(err)
		return nil, err
	}

	if dogestryCli.PullHosts != nil && len(dogestryCli.PullHosts) > 0 {
		var client *docker.Client
		for _, host := range dogestryCli.PullHosts {
			client, err = newDockerClient(host)
			if err != nil {
				log.Fatal(err)
				return nil, err
			}

			dogestryCli.PullClients = append(dogestryCli.PullClients, client)
		}
	} else {
		dogestryCli.PullHosts = []string{dogestryCli.DockerHost}
		dogestryCli.PullClients = []*docker.Client{dogestryCli.Client}
	}

	return dogestryCli, nil
}

type DogestryCli struct {
	Client      *docker.Client
	err         io.Writer
	TempDir     string
	TempDirRoot string
	DockerHost  string
	Config      config.Config
	PullHosts   []string
	PullClients []*docker.Client
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

func (cli *DogestryCli) Subcmd(name, signature, description string) *flag.FlagSet {
	flags := flag.NewFlagSet(name, flag.ContinueOnError)
	flags.Usage = func() {
		fmt.Fprintf(cli.err, "\nUsage: dogestry %s %s\n\n%s\n\n", name, signature, description)
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

func (cli *DogestryCli) getLayerIdsToDownload(fromId remote.ID, imageRoot string, r remote.Remote, client *docker.Client) ([]remote.ID, error) {
	toDownload := make([]remote.ID, 0)

	err := r.WalkImages(fromId, func(id remote.ID, image docker.Image, err error) error {
		fmt.Printf("Examining id '%s' on remote docker host...\n", id.Short())
		if err != nil {
			return err
		}

		_, err = client.InspectImage(string(id))

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
	toDownload, err := cli.getLayerIdsToDownload(fromId, imageRoot, r, cli.Client)
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

type Status struct {
	Host   string
	Status string
	Error  string `json:",omitempty"`
}

// makeStatusJSON returns status JSON
func (cli *DogestryCli) makeStatusJSON(errMap map[string]error) ([]byte, error) {
	var statusMap = make([]Status, len(cli.PullHosts))
	var status Status

	for i, host := range cli.PullHosts {
		status.Host = host

		if val, ok := errMap[host]; ok {
			status.Status = "failed"
			status.Error = val.Error()
		} else {
			status.Status = "ok"
		}

		statusMap[i] = status
	}
	return json.Marshal(statusMap)
}

func (cli *DogestryCli) outputStatus(errMap map[string]error) error {
	result, err := cli.makeStatusJSON(errMap)
	if err != nil {
		return err
	}
	fmt.Println(string(result[:]))
	return nil
}

// sendTar streams exported tarball into remote docker hosts
func (cli *DogestryCli) sendTar(imageRoot string) error {
	type hostErrTuple struct {
		host string
		err  error
	}

	type placeboProgressBar struct {
		count int
		bar   *pb.ProgressBar
	}

	progressBar := &placeboProgressBar{1000, pb.StartNew(1000)}
	progressBar.bar.ShowCounters = false

	// Starts the placebo progress bar
	go func(progressBar *placeboProgressBar) {
		for {
			progressBar.bar.Increment()
			time.Sleep(time.Second)
		}
	}(progressBar)

	tupleCh := make(chan hostErrTuple)

	for i, client := range cli.PullClients {
		host := cli.PullHosts[i]

		go func(client *docker.Client, host string) {
			cmd := exec.Command("tar", "cvf", "-", "-C", imageRoot, ".")
			cmd.Env = os.Environ()
			cmd.Dir = imageRoot
			defer cmd.Wait()

			stdout, err := cmd.StdoutPipe()
			if err != nil {
				tupleCh <- hostErrTuple{host, err}
				return
			}

			if err := cmd.Start(); err != nil {
				tupleCh <- hostErrTuple{host, err}
				return
			}

			err = client.LoadImage(docker.LoadImageOptions{InputStream: stdout})
			if err != nil {
				tupleCh <- hostErrTuple{host, err}
				return
			}
			tupleCh <- hostErrTuple{"", nil}

		}(client, host)
	}

	uploadImageErrMap := make(map[string]error)
	for range cli.PullClients {
		tuple := <-tupleCh
		if tuple.err != nil {
			uploadImageErrMap[tuple.host] = tuple.err
		}
	}
	close(tupleCh)

	err := cli.outputStatus(uploadImageErrMap)
	if len(uploadImageErrMap) > 0 {
		err = errors.New("Error in sendTar")
	}
	return err
}

type DownloadMap map[remote.ID][]string

func (cli *DogestryCli) makeDownloadMap(r remote.Remote, id remote.ID, imageRoot string) (DownloadMap, error) {
	var downloadMap = make(map[remote.ID][]string)
	var err error

	for i, pullHost := range cli.PullClients {
		fmt.Printf("Connecting to remote docker host: %v\n", cli.PullHosts[i])

		layers, err := cli.getLayerIdsToDownload(id, imageRoot, r, pullHost)
		if err != nil {
			return nil, err
		}

		for _, layer := range layers {
			downloadMap[layer] = append(downloadMap[layer], cli.PullHosts[i])
		}
	}
	return downloadMap, err
}

func (cli *DogestryCli) downloadImages(r remote.Remote, downloadMap DownloadMap, imageRoot string) error {
	pullImagesErrMap := make(map[string]error)

	for id, _ := range downloadMap {
		downloadPath := filepath.Join(imageRoot, string(id))

		fmt.Printf("Pulling image id '%s' to: %v\n", id.Short(), downloadPath)

		err := r.PullImageId(id, downloadPath)
		if err != nil {
			pullImagesErrMap[downloadPath] = err
		}
	}

	if len(pullImagesErrMap) > 0 {
		fmt.Printf("Errors pulling images: %v\n", pullImagesErrMap)
		return fmt.Errorf("Error downloading files from S3")
	}

	return nil
}
