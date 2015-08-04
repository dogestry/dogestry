package config

import (
	"os"
	"testing"
)

func TestNewConfig(t *testing.T) {
	os.Setenv("AWS_ACCESS_KEY_ID", "access")
	os.Setenv("AWS_SECRET_ACCESS_KEY", "secret")
	c, err := NewConfig(false)
	if err != nil {
		t.Fatalf("Failed to create config. Error: %v", err)
	}
	if c.AWS.AccessKeyID != "access" {
		t.Error("AccessKeyID should be 'access': " + c.AWS.AccessKeyID)
	}
	if c.AWS.SecretAccessKey != "secret" {
		t.Error("SecretAccessKey should be 'secret': " + c.AWS.SecretAccessKey)
	}
	if c.Docker.Connection == "" {
		t.Error("config.Docker.Connection should not be empty.")
	}

	os.Unsetenv("AWS_ACCESS_KEY_ID")
	os.Unsetenv("AWS_ACCESS_KEY")
	os.Unsetenv("AWS_SECRET_ACCESS_KEY")
	os.Unsetenv("AWS_SECRET_KEY")

	c, err = NewConfig(false)
	if err == nil || err.Error() != "AWS_ACCESS_KEY_ID/AWS_ACCESS_KEY or AWS_SECRET_ACCESS_KEY/AWS_SECRET_KEY are missing." {
		t.Error("should return error when evn vars are not set")
	}

	c, err = NewConfig(true)
	if err != nil {
		t.Error("should not renturn an error")
	}
}
