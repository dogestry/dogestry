package config

import (
	"os"

	"code.google.com/p/gcfg"
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
