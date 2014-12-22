package main

import (
	"flag"
	"fmt"
	"github.com/dogestry/dogestry/cli"
	"github.com/dogestry/dogestry/config"
	"log"
	"runtime"
)

var flConfigFile string
var flTempDir string

func init() {
	flag.StringVar(&flConfigFile, "config", "", "the dogestry config file (defaults to 'dogestry.cfg' in the current directory). Config is optional - if using s3 you can use env vars or signed URLs.")
	flag.StringVar(&flTempDir, "tempdir", "", "an alternate tempdir to use")
}

func main() {
	runtime.GOMAXPROCS(runtime.NumCPU())

	flag.Parse()

	args := flag.Args()

	cfg, err := config.NewConfig(flConfigFile)
	if err != nil {
		log.Fatal(err)
	}

	dogestryCli, err := cli.NewDogestryCli(cfg)
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
