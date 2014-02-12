package main

import (
	"flag"
	//"os"
	"github.com/blake-education/dogestry/cli"
  docker "github.com/blake-education/go-dockerclient"
	"log"
)

func main() {
  flConfigFile := flag.String("config", "", "the dogestry config file (defaults to 'dogestry.cfg' in the current directory)")
	flag.Parse()

  err := cli.ParseCommands(*flConfigFile, flag.Args()...)

	if err != nil {
		log.Println("err")
		log.Fatal(err)
	}
}
