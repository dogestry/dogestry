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
	outputChan     chan string
}

func NewProgressReader(r io.Reader, size int64, fileName string, outputChan chan string) io.Reader {
	return &ProgressReader{
		r:              r,
		TotalSize:      size,
		Current:        0,
		LastUpdate:     0,
		UpdateInterval: defaultInterval,
		FileName:       fileName,
		outputChan:     outputChan,
	}
}

func (p *ProgressReader) printProgress(progress, total int64, fileName string) {
	calc := fmt.Sprintf("%s/%s", HumanSize(progress), HumanSize(total))
	p.Print(fmt.Sprintf("  %-17s : %s", calc, fileName))
}

// Print messages to output channel (if available), otherwise via log.Print()
func (p *ProgressReader) Print(data ...string) {
	if p.outputChan != nil {
		for _, entry := range data {
			p.outputChan <- entry
		}
	} else {
		log.Println(data)
	}
}

func (p *ProgressReader) Read(in []byte) (n int, err error) {
	n, err = p.r.Read(in)
	p.Current += int64(n)

	if p.Current-p.LastUpdate > p.UpdateInterval {
		p.printProgress(p.Current, p.TotalSize, p.FileName)
		p.LastUpdate = p.Current
	}

	if err != nil {
		p.printProgress(p.Current, p.TotalSize, p.FileName)

		if err == io.EOF {
			p.Print(fmt.Sprintf("  %-17s : %s", "DONE", p.FileName))
		} else {
			p.Print(fmt.Sprintf("  %-17s: %s: %v", "ERROR", p.FileName, err))
		}
	}

	return
}
