package utils

import (
	"fmt"
	"io"
	"log"
	"os"
)

var progressLogger = log.New(os.Stdout, "", 0)

var defaultInterval int64 = 1024 * 1024 * 10 // 10 MB

type ProgressReader struct {
	r              io.Reader
	TotalSize      int64
	Current        int64
	LastUpdate     int64
	UpdateInterval int64
	FileName       string
}

func NewProgressReader(r io.Reader, size int64, fileName string) io.Reader {
	return &ProgressReader{r, size, 0, 0, defaultInterval, fileName}
}

func printProgress(progress, total int64, fileName string) {
	calc := fmt.Sprintf("%s/%s", HumanSize(progress), HumanSize(total))
	progressLogger.Printf("  %-17s : %s\n", calc, fileName)
}

func (p *ProgressReader) Read(in []byte) (n int, err error) {
	n, err = p.r.Read(in)
	p.Current += int64(n)

	if p.Current-p.LastUpdate > p.UpdateInterval {
		printProgress(p.Current, p.TotalSize, p.FileName)
		p.LastUpdate = p.Current
	}

	if err != nil {
		printProgress(p.Current, p.TotalSize, p.FileName)
		if err == io.EOF {
			progressLogger.Printf("  %-17s : %s\n", "DONE", p.FileName)
		} else {
			progressLogger.Printf("  %-17s: %s: %v\n", "ERROR", p.FileName, err)
		}
	}

	return
}
