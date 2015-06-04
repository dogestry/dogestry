package config

import (
	"testing"
)

func TestNewConfig(t *testing.T) {
	c, err := NewConfig()
	if err != nil {
		t.Fatalf("Failed to create config. Error: %v", err)
	}
	if c.Docker.Connection == "" {
		t.Error("config.Docker.Connection should not be empty.")
	}
}
