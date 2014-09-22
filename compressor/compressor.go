package compressor

import (
	"github.com/didip/dogestry/config"

	"fmt"
	"os"
	"os/exec"
	"strings"

	"io"
)

type Compressor struct {
	lz4Path string
}

func NewCompressor(config config.Config) (Compressor, error) {
	lz4 := config.Compressor.Lz4
	if lz4 == "" {
		lz4 = "lz4"
	}

	path, err := exec.LookPath(lz4)
	if err != nil {
		return Compressor{}, fmt.Errorf("can't find executable lz4 on the $PATH")
	}

	return Compressor{
		lz4Path: path,
	}, nil
}

// compress using lz4
// would use go version if we could (needs a streaming version)
// lz4 is low compression, but extremely fast
func (cmp Compressor) Compress(path string) error {
	compressedPath := path + ".lz4"

	err := exec.Command(cmp.lz4Path, path, compressedPath).Run()
	if err != nil {
		return err
	}

	return os.Remove(path)
}

func (cmp Compressor) CompressReader(r io.Reader) (out io.Reader, err error) {
	cmd := exec.Command(cmp.lz4Path, "-")

	cmd.Stdin = r
	out, err = cmd.StdoutPipe()
	if err != nil {
		return
	}

	err = cmd.Start()
	if err != nil {
		return
	}

	return
}

func (cmp Compressor) Decompress(path string) error {
	if !strings.HasSuffix(path, ".lz4") {
		return nil
	}

	uncompressedPath := strings.TrimSuffix(path, ".lz4")

	if _, err := os.Stat(path); !os.IsNotExist(err) {
		cmd := exec.Command(cmp.lz4Path, "-d", "-f", path, uncompressedPath)
		if err := cmd.Run(); err != nil {
			return err
		}

		return os.Remove(path)
	}

	return nil
}
