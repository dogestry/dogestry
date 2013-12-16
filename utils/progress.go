package utils

import (
  "fmt"
  "io"
)

type progressReader struct {
  r io.Reader
  TotalSize int64
  Output io.Writer
  Current int64
  LastUpdate int64
  UpdateInterval int64
}

func NewProgressReader(r io.Reader, size int64, w io.Writer) io.Reader {
  return &progressReader{r, size, w, 0, 0, 1024*512}
}


func printProgress(w io.Writer, progress, total int64) {
  fmt.Fprintf(w, "%s/%s         \r", HumanSize(progress), HumanSize(total))
}

func (p *progressReader) Read(in []byte) (n int, err error) {
  n,err = p.r.Read(in)
  p.Current += int64(n)

  if p.Current-p.LastUpdate > p.UpdateInterval {
    printProgress(p.Output, p.Current, p.TotalSize)
    p.LastUpdate = p.Current
  }


  if err != nil {
    printProgress(p.Output, p.Current, p.TotalSize)
    fmt.Fprintf(p.Output, "\n")
    if err == io.EOF {
      fmt.Fprintf(p.Output, "done\n")
    } else {
      fmt.Fprintf(p.Output, "error: %s\n", err)
    }
  }

  return
}
