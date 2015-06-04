package config

import (
	"errors"
	"net/url"
	"os"
)

func NewConfig() (Config, error) {
	c := Config{}
	c.AWS.AccessKeyID = os.Getenv("AWS_ACCESS_KEY_ID")
	c.AWS.SecretAccessKey = os.Getenv("AWS_SECRET_ACCESS_KEY")
	c.Docker.Connection = os.Getenv("DOCKER_HOST")

	if c.Docker.Connection == "" {
		c.Docker.Connection = "unix:///var/run/docker.sock"
	}
	if "" != os.Getenv("DOCKER_HOST") {
		c.Docker.Connection = os.Getenv("DOCKER_HOST")
	}

	if c.AWS.AccessKeyID == "" || c.AWS.SecretAccessKey == "" {
		return c, errors.New("AWS_ACCESS_KEY_ID or AWS_SECRET_ACCESS_KEY are missing.")
	}

	return c, nil
}

type Config struct {
	AWS struct {
		S3URL           *url.URL
		AccessKeyID     string
		SecretAccessKey string
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
