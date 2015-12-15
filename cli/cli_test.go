package cli

import (
	"os"
	"testing"

	"github.com/dogestry/dogestry/config"
)

const (
	testTmpDirRoot string = "/tmp"
)

var hosts = make([]string, 0)

func TestNewDogestryCli(t *testing.T) {
	cfg, err := config.NewConfig(false, 22375, false, false, false)
	if err != nil {
		t.Fatalf("Creating dogestry config should work. Error: %v", err)
	}

	dogestryCli, err := NewDogestryCli(cfg, hosts, testTmpDirRoot)
	if err != nil {
		t.Fatalf("Creating dogestryCli struct should work. Error: %v", err)
	}

	if dogestryCli.Client == nil {
		t.Fatal("dogestryCli.Client should never be nil.")
	}
}

func TestCreateAndReturnTempDirAndCleanup(t *testing.T) {
	cfg, err := config.NewConfig(false, 22375, false, false, false)
	if err != nil {
		t.Fatalf("Creating dogestry config should work. Error: %v", err)
	}

	dogestryCli, _ := NewDogestryCli(cfg, hosts, testTmpDirRoot)

	tmpDir, err := dogestryCli.CreateAndReturnTempDir()
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	if tmpDir == "" {
		t.Fatalf("CreateAndReturnTempDir should always return path to tmp directory. tmpDir: %v", tmpDir)
	}

	dogestryCli.Cleanup()
	if _, err := os.Stat(tmpDir); err == nil {
		t.Fatalf("Cleanup() should remove tmp directory. tmpDir: %v", tmpDir)
	}
}
