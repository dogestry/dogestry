package main

import (
	"flag"
	"fmt"
	"github.com/dogestry/dogestry/cli"
	"github.com/dogestry/dogestry/config"
	"log"
	"os"
	"runtime"
	"strings"
	"time"
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
	flLockFile   string
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
}

// getLock will return the lock file once it has exclusive access to it.
// This prevents multiple processes getting a lock at the same time.
func getLock(file string) (fp *os.File, err error) {
	for {
		fp, err = os.OpenFile(file, os.O_EXCL|os.O_CREATE|os.O_WRONLY, 0666)
		if patherr, ok := err.(*os.PathError); ok {
			if strings.Contains(patherr.Error(), "file exists") {
				// Lock file still exists, wait for a while and try again.
				time.Sleep(time.Second)
				continue
			}
		}
		// Either we suceeded creating the lock or an unknown error occured.
		return
	}
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

	if flLockFile != "" {
		log.Println("Waiting for lock file")
		if _, err := getLock(flLockFile); err != nil {
			log.Println("Lock error:", err)
			return
		}
		log.Println("Got lock")
		defer os.Remove(flLockFile)
	}

	args := flag.Args()

	cfg, err := config.NewConfig(flConfigFile)
	if err != nil {
		log.Fatal(err)
	}

	dogestryCli, err := cli.NewDogestryCli(cfg, flPullHosts)
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
