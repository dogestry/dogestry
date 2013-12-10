package cli

import (
	"dogestry/client"
	"dogestry/config"

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


type DogestryCli struct {
	client  client.Client
	err     io.Writer
	tempDir string
  Config  config.Config
}


func NewDogestryCli(client *client.Client, config config.Config) *DogestryCli {
	return &DogestryCli{
    Config: config,
		client: *client,
		err:    os.Stderr,
	}
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

func ParseCommands(configFilePath string, client *client.Client, args ...string) error {

  config,err := config.ParseConfig(configFilePath)
  if err != nil {
    return err
  }

	cli := NewDogestryCli(client, config)
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

