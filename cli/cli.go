package cli

import (
	docker "github.com/fsouza/go-dockerclient"
	"github.com/newrelic-forks/dogestry/config"

	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"reflect"
	"strings"
)

func ParseCommands(configFilePath string, tempDirRoot string, args ...string) error {
	cfg, err := config.NewConfig(configFilePath)
	if err != nil {
		return err
	}

	dogestryCli, err := NewDogestryCli(cfg)
	if err != nil {
		return err
	}
	defer dogestryCli.Cleanup()

	dogestryCli.tempDirRoot = tempDirRoot
	if dogestryCli.tempDirRoot == "" {
		dogestryCli.tempDirRoot = cfg.Dogestry.Temp_Dir
	}

	return dogestryCli.RunCmd(args...)

	return nil
}

func NewDogestryCli(cfg config.Config) (*DogestryCli, error) {
	var dockerHosts []string

	if cfg.HasMoreThanOneDockerHosts() {
		dockerHosts = cfg.GetDockerHosts()
	} else {
		dockerHosts = []string{cfg.GetDockerHost()}
	}

	fmt.Printf("Using docker endpoints: %v\n", dockerHosts)

	dogestryCli := &DogestryCli{
		Config:      cfg,
		err:         os.Stderr,
		DockerHosts: dockerHosts,
	}

	dogestryCli.Clients = make([]*docker.Client, 0)

	for _, dockerHost := range dockerHosts {
		newClient, err := docker.NewClient(dockerHost)
		if err != nil {
			return nil, err
		}
		dogestryCli.Clients = append(dogestryCli.Clients, newClient)
	}

	return dogestryCli, nil
}

type DogestryCli struct {
	Clients     []*docker.Client
	err         io.Writer
	tempDir     string
	tempDirRoot string
	DockerHosts []string
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
	 dogestry pull s3://<bucket name>/<path name>/?region=us-east-1 <repo name>
  Commands:
	 pull - Pull an image from a remote
	 push  - Push an image to a remote
	 remote - Check a remote
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

// Creates and returns temporary work dir
// This dir is cleaned up on exit
func (cli *DogestryCli) TempDir() string {
	if cli.tempDir == "" {
		if cli.tempDirRoot != "" {
			if err := os.MkdirAll(cli.tempDirRoot, 0755); err != nil {
				log.Fatal(err)
			}
		}

		if tempDir, err := ioutil.TempDir(cli.tempDirRoot, "dogestry"); err != nil {
			log.Fatal(err)
		} else {
			cli.tempDir = tempDir
		}
	}

	return cli.tempDir
}

// Creates and returns a workdir under TempDir
func (cli *DogestryCli) WorkDir(suffix string) (string, error) {
	suffix = strings.Replace(suffix, ":", "_", -1)

	path := filepath.Join(cli.TempDir(), suffix)

	fmt.Printf("WorkDir: %v\n", path)

	if err := os.MkdirAll(path, os.ModeDir|0700); err != nil {
		return "", err
	}

	return path, nil
}

// clean up the tempDir
func (cli *DogestryCli) Cleanup() {
	if cli.tempDir != "" {
		if err := os.RemoveAll(cli.tempDir); err != nil {
			log.Println(err)
		}
	}
}
