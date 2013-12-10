package main

import (
	"flag"
	//"os"
	"dogestry/cli"
	"dogestry/client"
	"log"
)




func main() {
  flConfigFile := flag.String("config", "", "the dogestry config file (defaults to 'dogestry.cfg' in the current directory)")
	flag.Parse()


	client, err := client.NewClient("unix:///var/run/docker.sock")
	if err != nil {
		log.Fatal(err)
	}

	if err = cli.ParseCommands(*flConfigFile, client, flag.Args()...); err != nil {
		log.Println("err")
		log.Fatal(err)
	}
}
