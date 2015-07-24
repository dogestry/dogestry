package config

import (
	"errors"
	"net/url"
	"os"
)

func NewConfig(useMetaService bool) (Config, error) {
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

	if !useMetaService && (c.AWS.AccessKeyID == "" || c.AWS.SecretAccessKey == "") {
		return c, errors.New("AWS_ACCESS_KEY_ID/AWS_ACCESS_KEY or AWS_SECRET_ACCESS_KEY/AWS_SECRET_KEY are missing.")
	}

	return c, nil
}

type Config struct {
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
