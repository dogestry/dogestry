package compressor

import (
  "os/exec"
)

type Compressor struct {}

func NewCompressor(config config.Config) Compressor {
}



// compress using lz4
// would use go version if we could (needs a streaming version)
// lz4 is low compression, but extremely fast
func (cmp *Compressor) compress(path string) error {
  err := exec.Command("./lz4", path, path+".lz4").Run()
  if err != nil {
    return err
  }

  return os.Remove(path)
}


decompress(path string) error {

compressedLayerFile := filepath.Join(dst, "layer.tar.lz4")
  layerFile := filepath.Join(dst, "layer.tar")

  if _, err := os.Stat(compressedLayerFile); !os.IsNotExist(err) {
    fmt.Println("exists?", compressedLayerFile)
    cmd := exec.Command("./lz4", "-d", "-f", compressedLayerFile, layerFile)
    if err := cmd.Run(); err != nil {
      return err
    }

    return os.Remove(compressedLayerFile)
  }


