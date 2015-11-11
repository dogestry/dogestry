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
	"github.com/dogestry/dogestry/server"
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
	flServerMode     bool
	flServerAddress  string
	flServerPort     int
	flForceLocal     bool
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
	flag.BoolVar(&flServerMode, "server", false, "run dogestry in server mode")
	flag.StringVar(&flServerAddress, "address", "0.0.0.0", "what address to bind to when running dogestry in server mode")
	flag.IntVar(&flServerPort, "port", 22375, "what port to bind to when running dogestry in server mode")
	flag.BoolVar(&flForceLocal, "force-local", false, "do not try to use the dogestry server on host endpoints")
}

func main() {
	runtime.GOMAXPROCS(runtime.NumCPU())

	flag.Usage = func() {
		fmt.Fprintln(os.Stderr, cli.HelpMessage)
	}

	flag.Parse()

	if flVersion {
		if err := cli.PrintVersion(); err != nil {
			log.Fatal(err)
		}
		return
	}

	if flServerMode {
		fullAddress := fmt.Sprintf("%v:%v", flServerAddress, flServerPort)

		log.Printf("Running dogestry in server mode on '%v'", fullAddress)

		s := server.New(fullAddress)
		s.ServeHttp()
	} else {
		args := flag.Args()

		// Allow 'help', 'version' and 'login' to not require AWS cred env vars
		requireEnvVars := true

		if len(args) == 0 || (args[0] == "help" || args[0] == "login" || args[0] == "version") {
			requireEnvVars = false
		}

		cfg, err := config.NewConfig(flUseMetaService, flServerPort, flForceLocal, requireEnvVars)
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
}
