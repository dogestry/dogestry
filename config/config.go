package config

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"os"
)

func NewConfig(useMetaService bool, serverPort int, forceLocal, requireEnvVars bool) (Config, error) {
	c := Config{}
	c.AWS.AccessKeyID = os.Getenv("AWS_ACCESS_KEY_ID")
	if c.AWS.AccessKeyID == "" {
		c.AWS.AccessKeyID = os.Getenv("AWS_ACCESS_KEY")
	}

	c.AWS.SecretAccessKey = os.Getenv("AWS_SECRET_ACCESS_KEY")
	if c.AWS.SecretAccessKey == "" {
		c.AWS.SecretAccessKey = os.Getenv("AWS_SECRET_KEY")
	}

	c.Docker.Connection = os.Getenv("DOCKER_HOST")

	if c.Docker.Connection == "" {
		c.Docker.Connection = "unix:///var/run/docker.sock"
	}

	c.AWS.UseMetaService = useMetaService

	if requireEnvVars {
		if !useMetaService && (c.AWS.AccessKeyID == "" || c.AWS.SecretAccessKey == "") {
			return c, errors.New("AWS_ACCESS_KEY_ID/AWS_ACCESS_KEY or AWS_SECRET_ACCESS_KEY/AWS_SECRET_KEY are missing.")
		}
	}

	c.ServerPort = serverPort
	c.ForceLocal = forceLocal

	return c, nil
}

// Config instantiation when dogestry is ran in server mode
func NewServerConfig(authHeader string) (Config, error) {
	c := Config{}

	data, err := base64.StdEncoding.DecodeString(authHeader)
	if err != nil {
		return c, fmt.Errorf("Unbale to base64 decode auth header: %v", err)
	}

	var authConfig AuthConfig

	if err := json.Unmarshal(data, &authConfig); err != nil {
		return c, fmt.Errorf("Unable to unmarshal JSON authconfig: %v", err)
	}

	if authConfig.Username == "" {
		return c, errors.New("Missing username/AccessKeyID in auth header")
	} else if authConfig.Password == "" {
		return c, errors.New("Missing password/SecretAccessKey in auth header")
	} else if authConfig.Email == "" {
		return c, errors.New("Missing email/S3Bucket in auth header")
	}

	if err := c.SetS3URL(authConfig.Email); err != nil {
		return c, fmt.Errorf("Unable to set S3URL: %v", err)
	}

	c.AWS.AccessKeyID = authConfig.Username
	c.AWS.SecretAccessKey = authConfig.Password

	c.Docker.Connection = os.Getenv("DOCKER_HOST")

	if c.Docker.Connection == "" {
		c.Docker.Connection = "unix:///var/run/docker.sock"
	}

	c.ServerMode = true

	return c, nil
}

type AuthConfig struct {
	Username      string `json:"username,omitempty"`
	Password      string `json:"password,omitempty"`
	Auth          string `json:"auth"`
	Email         string `json:"email"`
	ServerAddress string `json:"serveraddress,omitempty"`
}

type Config struct {
	ServerMode bool
	ServerPort int
	ForceLocal bool // whether to attempt remote dogestry server usage

	AWS struct {
		S3URL           *url.URL
		AccessKeyID     string
		SecretAccessKey string
		UseMetaService  bool
	}
	Docker struct {
		Connection string
	}
}

func (c *Config) SetS3URL(rawurl string) error {
	urlStruct, err := url.Parse(rawurl)
	if err != nil {
		return err
	}

	c.AWS.S3URL = urlStruct

	return nil
}
