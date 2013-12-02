package cli

import (
  "dogestry/client"
  "fmt"
  "flag"
  "io"
  "io/ioutil"
  "log"
  "os"
  "reflect"
  "strings"
)

// snatched from docker
func (cli *DogestryCli) getMethod(name string) (func(...string) error, bool) {
	methodName := "Cmd" + strings.ToUpper(name[:1]) + strings.ToLower(name[1:])
	method := reflect.ValueOf(cli).MethodByName(methodName)
	if !method.IsValid() {
		return nil, false
	}
	return method.Interface().(func(...string) error), true
}

func ParseCommands(client *client.Client, args ...string) error {
	cli := NewDogestryCli(client)
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


func (cli *DogestryCli) TempDir() string {
  if cli.tempDir == "" {
    if tempDir,err := ioutil.TempDir("","dogestry"); err != nil {
      log.Fatal(err)
    } else {
      cli.tempDir = tempDir
    }
  }

  return cli.tempDir
}


func (cli *DogestryCli) Cleanup() {
  fmt.Println("cleaning up", cli.tempDir)
  if cli.tempDir != "" {
    if err := os.RemoveAll(cli.tempDir); err != nil {
      log.Println(err)
    }
  }
}


func NewDogestryCli(client *client.Client) *DogestryCli {
  return &DogestryCli{
    client: *client,
    err: os.Stderr,
  }
}


type DogestryCli struct {
	client     client.Client
  err        io.Writer
  tempDir    string
}
