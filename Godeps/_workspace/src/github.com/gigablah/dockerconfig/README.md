DockerConfig
============

This package is essentially a fork of `github.com/docker/docker/cliconfig`.

The main difference is the ability to manage both legacy `.dockercfg` and new `config.json` (Docker 1.7+) files.

Migrations between both formats can be easily done by changing the version number.

Usage
-----

```go
package foo

import (
	"fmt"

	"github.com/gigablah/dockerconfig"
)

func main() {
	// Try to locate a Docker config file
	// Looks for ~/.docker/config.json and falls back to ~/.dockercfg
	// Base directory can be changed or read from DOCKER_CONFIG
	if config, err := dockerconfig.Load(""); err != nil {
		panic(fmt.Errorf("%v", err))
	}

	fmt.Printf("Writing to Docker file %v...\n", config.Filename())

	auth, ok := config.AuthConfigs["example.com"]
	if !ok {
		auth = dockerconfig.AuthConfig{}
	}
	auth.Username = "Foo"
	auth.Password = "Bar"
	auth.Email = "foobar@example.com"
	auth.ServerAddress = "example.com"
	config.AuthConfigs["example.com"] = auth

	// Save Docker config back to the same file it was loaded from
	if err := config.Save(); err != nil {
		panic(fmt.Errorf("%v", err))
	}

	// Save Docker config in the new config.json location and format
	config.version = 2
	err := config.Save()
}
```


License
-------

Released under the Apache License, Version 2.0.
