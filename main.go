package main

import (
	"flag"
  //"launchpad.net/goamz/s3"
  //"os"
  "log"
  "dogestry/cli"
  "dogestry/client"
)


func main() {
	flag.Parse()

  client,err := client.NewClient("unix:///var/run/docker.sock")
  if err != nil {
    log.Fatal(err)
  }

  cli.ParseCommands(client, flag.Args()...)

  log.Println("ok, done")
}
