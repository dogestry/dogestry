package main

import (
	"flag"
	"fmt"
	"github.com/dogestry/dogestry/cli"
	"github.com/dogestry/dogestry/config"
	"log"
	"runtime"
	"strings"
)

type pullHosts []string

func (h *pullHosts) String() string {
	return fmt.Sprintf("%v", *h)
}

func (h *pullHosts) Set(value string) error {
	for _, host := range strings.Split(value, ",") {
		*h = append(*h, host)
	}
	return nil
}

var flConfigFile string
var flTempDir string
var flPullHosts pullHosts

func init() {
	flag.StringVar(&flConfigFile, "config", "", "the dogestry config file (defaults to 'dogestry.cfg' in the current directory). Config is optional - if using s3 you can use env vars or signed URLs.")
	flag.StringVar(&flTempDir, "tempdir", "", "an alternate tempdir to use")
	flag.Var(&flPullHosts, "pullhosts", "a comma-separated list of docker hosts where the image will be pulled")
}

func main() {
	runtime.GOMAXPROCS(runtime.NumCPU())

	flag.Parse()

	args := flag.Args()

	cfg, err := config.NewConfig(flConfigFile)
	if err != nil {
		log.Fatal(err)
	}

	dogestryCli, err := cli.NewDogestryCli(cfg, flPullHosts)
	if err != nil {
		log.Fatal(err)
	}

	dogestryCli.TempDirRoot = flTempDir
	if dogestryCli.TempDirRoot == "" {
		dogestryCli.TempDirRoot = cfg.Dogestry.Temp_Dir
	}

	err = dogestryCli.RunCmd(args...)

	if err == nil {
		if len(args) > 0 && args[0] == "download" {
			fmt.Printf("%v\n", dogestryCli.TempDir)
		} else {
			dogestryCli.Cleanup()
		}
	} else {
		dogestryCli.Cleanup()
		log.Fatal(err)
	}
}
