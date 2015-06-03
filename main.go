package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"runtime"
	"strings"

	"github.com/dogestry/dogestry/cli"
	"github.com/dogestry/dogestry/config"
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

var (
	flConfigFile string
	flVersion    bool
	flPullHosts  pullHosts
)

func init() {
	const (
		versionDefault = false
		versionUsage   = "print version"
	)

	flag.StringVar(&flConfigFile, "config", "", "the dogestry config file (defaults to 'dogestry.cfg' in the current directory). Config is optional - if using s3 you can use env vars or signed URLs.")
	flag.BoolVar(&flVersion, "version", versionDefault, versionUsage)
	flag.BoolVar(&flVersion, "v", versionDefault, versionUsage+" (short)")
	flag.Var(&flPullHosts, "pullhosts", "a comma-separated list of docker hosts where the image will be pulled")
}

func main() {
	runtime.GOMAXPROCS(runtime.NumCPU())

	flag.Usage = func() {
		fmt.Fprintln(os.Stderr, cli.HelpMessage)
	}

	flag.Parse()

	if flVersion {
		err := cli.PrintVersion()
		if err != nil {
			log.Fatal(err)
		}
		return
	}

	args := flag.Args()

	dogestryCli, err := cli.NewDogestryCli(config.NewConfig(), flPullHosts)
	if err != nil {
		log.Fatal(err)
	}

	err = dogestryCli.RunCmd(args...)

	if err == nil {
		dogestryCli.Cleanup()
	} else {
		dogestryCli.Cleanup()
		log.Fatal(err)
	}
}
