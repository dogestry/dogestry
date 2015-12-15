package utils

import (
	"bufio"
	"crypto/md5"
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
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

// Extract hostnames and ports from a list of pullhost args
// ie. 'tcp://some.hostname.com:2375', extract 'some.hostname.com' and '2375'
//
// return map[hostname]port
func ParseHosts(pullHosts []string) map[string]int {
	parsedHosts := make(map[string]int, 0)

	for _, hostEntry := range pullHosts {
		parsed, err := url.Parse(hostEntry)
		if err != nil {
			continue
		}

		if parsed.Scheme == "tcp" && parsed.Host != "" {
			splitHost := strings.Split(parsed.Host, ":")
			port, err := strconv.ParseInt(splitHost[1], 10, 64)
			if err != nil {
				continue
			}

			parsedHosts[splitHost[0]] = int(port)
		}
	}

	return parsedHosts
}

// Check if docker (or dogestry) is running on the endpoint
func ServerCheck(host string, port int, timeout time.Duration, docker bool) bool {
	url := fmt.Sprintf("http://%v:%v/version", host, port)

	if !docker {
		url = fmt.Sprintf("http://%v:%v/status/check", host, port)
	}

	client := http.Client{
		Timeout: timeout,
	}

	resp, getErr := client.Get(url)
	if getErr != nil {
		return false
	}
	defer resp.Body.Close()

	if docker {
		if resp.StatusCode != 200 {
			return false
		}
	} else {
		// Dogestry check
		body, readErr := ioutil.ReadAll(resp.Body)
		if readErr != nil {
			return false
		}

		if string(body) != "OK" {
			return false
		}
	}

	return true
}
