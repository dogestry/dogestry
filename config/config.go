package config

import (
	"code.google.com/p/gcfg"
	"fmt"
	"os"
	"strings"
)

var (
	DefaultConfigFilePath = "./dogestry.cfg"
	DefaultConfig         = Config{
		Remote: make(map[string]*RemoteConfig),
	}
)

func NewConfig(configFilePath string) (config Config, err error) {
	if configFilePath == "" {
		if _, err := os.Stat(DefaultConfigFilePath); !os.IsNotExist(err) {
			configFilePath = DefaultConfigFilePath
		} else {
			fmt.Fprintln(os.Stdout, "Note: no config file found, using default config.")
			return DefaultConfig, nil
		}
	}

	err = gcfg.ReadFileInto(&config, configFilePath)
	return
}

type RemoteConfig struct {
	Url string
}

type S3Config struct {
	Access_Key_Id string
	Secret_Key    string
}

type DockerConfig struct {
	Connection string
}

type DogestryConfig struct {
	Temp_Dir string
}

type Config struct {
	Remote   map[string]*RemoteConfig
	S3       S3Config
	Docker   DockerConfig
	Dockers  []DockerConfig
	Dogestry DogestryConfig
}

func (c *Config) GetDockerHost() string {
	dockerHost := c.Docker.Connection

	if "" != os.Getenv("DOCKER_HOST") {
		dockerHost = os.Getenv("DOCKER_HOST")
	}

	if "" == dockerHost {
		dockerHost = "tcp://localhost:2375"
	}
	return dockerHost
}

func (c *Config) GetDockerHosts() []string {
	var dockerHosts []string

	// Environment Variable takes higher precedence.
	if "" != os.Getenv("DOCKER_HOSTS") {
		dockerHosts = strings.Split(os.Getenv("DOCKER_HOSTS"), ",")
		for i, dockerHost := range dockerHosts {
			dockerHosts[i] = strings.TrimSpace(dockerHost)
		}
		return dockerHosts
	}

	for _, docker := range c.Dockers {
		dockerHosts = append(dockerHosts, docker.Connection)
	}

	if len(dockerHosts) == 0 {
		dockerHosts = []string{"tcp://localhost:2375"}
	}

	return dockerHosts
}

func (c *Config) HasMoreThanOneDockerHosts() bool {
	if len(c.GetDockerHosts()) > 0 {
		return true
	} else {
		return false
	}
}
