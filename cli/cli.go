package cli

import (
	docker "github.com/fsouza/go-dockerclient"
	"github.com/didip/dogestry/config"

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

var (
	DefaultConfigFilePath = "./dogestry.cfg"
	DefaultConfig         = config.Config{
		Remote: make(map[string]*config.RemoteConfig),
	}
)

type DogestryCli struct {
	client      docker.Client
	err         io.Writer
	tempDir     string
	tempDirRoot string
	Config      config.Config
}

func NewDogestryCli(config config.Config) (*DogestryCli, error) {
	dockerConnection := config.Docker.Connection

	if "" != os.Getenv("DOCKER_HOST") {
		dockerConnection = os.Getenv("DOCKER_HOST")
	}

	if "" == dockerConnection {
		dockerConnection = "tcp://localhost:2375"
	} else {
		fmt.Println("Docker connection set from file")
	}

	fmt.Printf("Using docker endpoint: [%s]\n", dockerConnection)

	newClient, err := docker.NewClient(dockerConnection)
	if err != nil {
		log.Fatal(err)
	}

	return &DogestryCli{
		Config: config,
		client: *newClient,
		err:    os.Stderr,
	}, nil
}

// Note: snatched from docker

func (cli *DogestryCli) getMethod(name string) (func(...string) error, bool) {
	methodName := "Cmd" + strings.ToUpper(name[:1]) + strings.ToLower(name[1:])
	method := reflect.ValueOf(cli).MethodByName(methodName)
	if !method.IsValid() {
		return nil, false
	}
	return method.Interface().(func(...string) error), true
}

func ParseCommands(configFilePath string, tempDirRoot string, args ...string) error {
	config, err := parseConfig(configFilePath)
	if err != nil {
		return err
	}

	cli, err := NewDogestryCli(config)
	if err != nil {
		return err
	}
	defer cli.Cleanup()

	cli.tempDirRoot = tempDirRoot
	if cli.tempDirRoot == "" {
		cli.tempDirRoot = config.Dogestry.Temp_Dir
	}

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

func parseConfig(configFilePath string) (cfg config.Config, err error) {
	// no config file was specified
	if configFilePath == "" {
		// if default config exists use it
		if _, err := os.Stat(DefaultConfigFilePath); !os.IsNotExist(err) {
			configFilePath = DefaultConfigFilePath
		} else {
			fmt.Fprintln(os.Stdout, "Note: no config file found, using default config.")
			return DefaultConfig, nil
		}
	}

	return config.ParseConfig(configFilePath)
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
	path := filepath.Join(cli.TempDir(), strings.Replace(suffix, ":", "_", -1))

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
