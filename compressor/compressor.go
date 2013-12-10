package compressor

import (
  "os/exec"
)

type Compressor struct {
  lz4Path string
}


func NewCompressor(config config.Config) (Compressor, error) {
  path,err := filepath.LookPath(config.Compressor.Lz4)
  if err != nil {
    return Compressor{}, err
  }

  return Compressor{
    lz4Path: path,
  }
}



// compress using lz4
// would use go version if we could (needs a streaming version)
// lz4 is low compression, but extremely fast
func (cmp *Compressor) compress(path string) error {
  err := exec.Command(cmp.lz4Path, path, path+".lz4").Run()
  if err != nil {
    return err
  }

  return os.Remove(path)
}


func (cmp *Compressor) decompress(path string) error {
  layerFile := filepath.Join(filepath.Dir(path), )

  if _, err := os.Stat(compressedLayerFile); !os.IsNotExist(err) {
    fmt.Println("exists?", compressedLayerFile)
    cmd := exec.Command(cmp.lz4Path, "-d", "-f", compressedLayerFile, layerFile)
    if err := cmd.Run(); err != nil {
      return err
    }

    return os.Remove(compressedLayerFile)
  }
}
