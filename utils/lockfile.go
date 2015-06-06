package utils

import (
	"log"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"
)

type DogestryCliLike interface {
	RunCmd(...string) error
	Cleanup()
}

func LockByFile(dogestryCli DogestryCliLike, args []string, lockfile string) {
	signalc := make(chan os.Signal, 1)
	signal.Notify(signalc, os.Interrupt, os.Kill, syscall.SIGTERM)

	lockerrc := make(chan error)
	locked := make(chan struct{})
	if lockfile != "" {
		log.Println("Waiting for lock file")
		go func() {
			lockerrc <- getLock(lockfile)
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
			defer os.Remove(lockfile)
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
