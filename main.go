package main

import (
	"flag"
	//"launchpad.net/goamz/s3"
	//"os"
	"dogestry/cli"
	"dogestry/client"
	"log"
)

func main() {
	flag.Parse()

	client, err := client.NewClient("unix:///var/run/docker.sock")
	if err != nil {
		log.Fatal(err)
	}

	if err = cli.ParseCommands(client, flag.Args()...); err != nil {
    log.Println("err")
		log.Fatal(err)
	}

	log.Println("ok, done")
}
