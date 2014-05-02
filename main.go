package main

import (
	"flag"
	//"os"
	"log"

	"github.com/blake-education/dogestry/cli"
)

func main() {
	flConfigFile := flag.String("config", "", "the dogestry config file (defaults to 'dogestry.cfg' in the current directory). Config is optional - if using s3 you can use env vars or signed URLs.")
	flTempDir := flag.String("tempdir", "", "an alternate tempdir to use")
	flag.Parse()

	err := cli.ParseCommands(*flConfigFile, *flTempDir, flag.Args()...)

	if err != nil {
		log.Println("err")
		log.Fatal(err)
	}
}
