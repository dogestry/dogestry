package utils

import (
	"bufio"
	"crypto/md5"
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"io"
	"net"
	"os"
	"regexp"
	"time"
)

// HumanSize returns a human-readable approximation of a size
// using SI standard (eg. "44kB", "17MB")
func HumanSize(size int64) string {
	i := 0
	var sizef float64
	sizef = float64(size)
	units := []string{"B", "kB", "MB", "GB", "TB", "PB", "EB", "ZB", "YB"}
	for sizef >= 1000.0 {
		sizef = sizef / 1000.0
		i++
	}
	return fmt.Sprintf("%.4g %s", sizef, units[i])
}

func FileHumanSize(path string) string {
	var size int64
	info, err := os.Stat(path)
	if err != nil {
		size = 0
	} else {
		size = info.Size()
	}

	return HumanSize(size)
}

// md5 file at path
func Md5File(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", nil
	}
	defer f.Close()

	// files could be pretty big, lets buffer
	buff := bufio.NewReader(f)
	hash := md5.New()

	io.Copy(hash, buff)
	return hex.EncodeToString(hash.Sum(nil)), nil
}

// sha1 file at path
func Sha1File(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", nil
	}
	defer f.Close()

	// files could be pretty big, lets buffer
	buff := bufio.NewReader(f)
	hash := sha1.New()

	io.Copy(hash, buff)
	return hex.EncodeToString(hash.Sum(nil)), nil
}

// Extract hostnames from a list of pullhost args
// ie. 'tcp://some.hostname.com:2375', extract 'some.hostname.com'
func ParseHostnames(pullHosts []string) []string {
	parsedHosts := make([]string, 0)

	re := regexp.MustCompile("^tcp://(.+):2375$")

	for _, hostEntry := range pullHosts {
		results := re.FindSubmatch([]byte(hostEntry))

		if len(results) == 2 {
			parsedHosts = append(parsedHosts, string(results[1]))
		}
	}

	return parsedHosts
}

// See if remote Dogestry port is open
func DogestryServerCheck(host string, port int, timeout time.Duration) bool {
	conn, err := net.DialTimeout("tcp", fmt.Sprintf("%v:%v", host, port), timeout)
	if err != nil {
		return false
	}

	conn.Close()

	return true
}
