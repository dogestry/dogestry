package utils

import (
	"io"
	"log"
	"os"
)

var progressLogger = log.New(os.Stdout, "", 0)

type progressReader struct {
	r              io.Reader
	TotalSize      int64
	Current        int64
	LastUpdate     int64
	UpdateInterval int64
	FileName       string
}

func NewProgressReader(r io.Reader, size int64, fileName string) io.Reader {
	return &progressReader{r, size, 0, 0, 1024 * 512, fileName}
}

func printProgress(progress, total int64, fileName string) {
	progressLogger.Printf("%s: %s/%s\n", fileName, HumanSize(progress), HumanSize(total))
}

func (p *progressReader) Read(in []byte) (n int, err error) {
	n, err = p.r.Read(in)
	p.Current += int64(n)

	if p.Current-p.LastUpdate > p.UpdateInterval {
		printProgress(p.Current, p.TotalSize, p.FileName)
		p.LastUpdate = p.Current
	}

	if err != nil {
		printProgress(p.Current, p.TotalSize, p.FileName)
		if err == io.EOF {
			progressLogger.Printf("done\n")
		} else {
			progressLogger.Printf("error: %s\n", err)
		}
	}

	return
}
