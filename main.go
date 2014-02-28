package main

import (
	"flag"
	//"os"
	"github.com/blake-education/dogestry/cli"
	"log"
)

func main() {
  flConfigFile := flag.String("config", "", "the dogestry config file (defaults to 'dogestry.cfg' in the current directory). Config is optional - if using s3 you can use env vars or signed URLs.")
	flag.Parse()

  err := cli.ParseCommands(*flConfigFile, flag.Args()...)

	if err != nil {
		log.Println("err")
		log.Fatal(err)
	}
}
