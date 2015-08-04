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
	"github.com/dogestry/dogestry/utils"
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
	flConfigFile     string
	flVersion        bool
	flPullHosts      pullHosts
	flLockFile       string
	flUseMetaService bool
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
	flag.StringVar(&flLockFile, "lockfile", "", "lockfile to use while executing command, prevents parallel executions")
	flag.BoolVar(&flUseMetaService, "use-metaservice", false, "use tha AWS metadata service to get credentials")
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

	cfg, err := config.NewConfig(flUseMetaService)
	if err != nil {
		log.Fatal(err)
	}

	dogestryCli, err := cli.NewDogestryCli(cfg, flPullHosts)
	if err != nil {
		log.Fatal(err)
	}

	if flLockFile != "" {
		utils.LockByFile(dogestryCli, args, flLockFile)
	} else {
		err = dogestryCli.RunCmd(args...)

		dogestryCli.Cleanup()

		if err != nil {
			log.Fatal(err)
		}
	}
}
