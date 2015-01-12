package cli

import (
	"github.com/dogestry/dogestry/config"
	"os"
	"testing"
)

var hosts = make([]string, 0)

func TestNewDogestryCli(t *testing.T) {
	cfg, err := config.NewConfig("")
	if err != nil {
		t.Fatalf("Creating config struct should work. Error: %v", err)
	}

	dogestryCli, err := NewDogestryCli(cfg, hosts)
	if err != nil {
		t.Fatalf("Creating dogestryCli struct should work. Error: %v", err)
	}

	if dogestryCli.Client == nil {
		t.Fatal("dogestryCli.Client should never be nil.")
	}
}

func TestCreateAndReturnTempDirAndCleanup(t *testing.T) {
	cfg, _ := config.NewConfig("")
	dogestryCli, _ := NewDogestryCli(cfg, hosts)

	tmpDir := dogestryCli.CreateAndReturnTempDir()
	if tmpDir == "" {
		t.Fatalf("CreateAndReturnTempDir should always return path to tmp directory. tmpDir: %v", tmpDir)
	}

	dogestryCli.Cleanup()
	if _, err := os.Stat(tmpDir); err == nil {
		t.Fatalf("Cleanup() should remove tmp directory. tmpDir: %v", tmpDir)
	}
}
