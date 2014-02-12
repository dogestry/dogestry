package cli

import (
  docker "github.com/blake-education/go-dockerclient"
	"github.com/blake-education/dogestry/config"

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
  DefaultConfig = config.Config{
    Remote: make(map[string]*config.RemoteConfig),
    Compressor: config.CompressorConfig{
      Lz4: "lz4",
    },
  }
)



type DogestryCli struct {
	client  docker.Client
	err     io.Writer
	tempDir string
  Config  config.Config
}


func NewDogestryCli(config config.Config) (*DogestryCli,error) {
  dockerConnection := config.Docker.Connection
  if dockerConnection == "" {
    dockerConnection = "unix:///var/run/docker.sock"
  }

	newClient, err := client.NewClient(dockerConnection)
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

func ParseCommands(configFilePath string, args ...string) error {
  config,err := parseConfig(configFilePath)
  if err != nil {
    return err
  }

	cli,err := NewDogestryCli(config)
  if err != nil {
    return err
  }
	defer cli.Cleanup()

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


func parseConfig(configFilePath string) (cfg config.Config,err error) {
  // no config file was specified
  if configFilePath == "" {
    // if default config exists use it
    if _,err := os.Stat(DefaultConfigFilePath); !os.IsNotExist(err) {
      configFilePath = DefaultConfigFilePath
    } else {
      fmt.Fprintln(os.Stderr, "Warning: no config file found, using default config")
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

	fmt.Println("you are beyond help", args)
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
		if tempDir, err := ioutil.TempDir("", "dogestry"); err != nil {
			log.Fatal(err)
		} else {
			cli.tempDir = tempDir
		}
	}

	return cli.tempDir
}

// Creates and returns a workdir under TempDir
func (cli *DogestryCli) WorkDir(suffix string) (string, error) {
	path := filepath.Join(cli.TempDir(), suffix)

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

