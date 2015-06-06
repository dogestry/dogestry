package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"runtime"
	"strings"
	"syscall"
	"time"

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
func getLock(file string) error {
	for {
		_, err := os.OpenFile(file, os.O_EXCL|os.O_CREATE|os.O_WRONLY, 0666)
		if patherr, ok := err.(*os.PathError); ok {
			if strings.Contains(patherr.Error(), "file exists") {
				// Lock file still exists, wait for a while and try again.
				time.Sleep(time.Second)
				continue
			}
		}
		// Either we suceeded creating the lock or an unknown error occured.
		return err
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

	args := flag.Args()

	cfg, err := config.NewConfig()
	if err != nil {
		log.Fatal(err)
	}

	dogestryCli, err := cli.NewDogestryCli(cfg, flPullHosts)
	if err != nil {
		log.Fatal(err)
	}

	signalc := make(chan os.Signal, 1)
	signal.Notify(signalc, os.Interrupt, os.Kill, syscall.SIGTERM)

	lockerrc := make(chan error)
	locked := make(chan struct{})
	if flLockFile != "" {
		log.Println("Waiting for lock file")
		go func() {
			lockerrc <- getLock(flLockFile)
		}()
	} else {
		close(locked)
	}

	errc := make(chan error)
	go func() {
		<-locked
		errc <- dogestryCli.RunCmd(args...)
	}()

	for {
		select {
		case err := <-lockerrc:
			if err != nil {
				log.Println(err)
				return
			}
			defer os.Remove(flLockFile)
			close(locked)
			// We don't expect more than one error so disable this channel
			lockerrc = nil
		case err := <-errc:
			if err != nil {
				log.Println(err)
			}
			dogestryCli.Cleanup()
			return
		case <-signalc:
			log.Println("Got signal, exiting")
			return
			// TODO: Also make it possible for dogestry cli to cancel pending actions.
		}
	}
}
