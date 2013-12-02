package remote

import (
  "errors"
  "fmt"
	"net/url"
)

var (
	// ErrInvalidRemote is returned when the remote is not a valid.
	ErrInvalidRemote = errors.New("Invalid endpoint")
)



func Push(remote, image, imageRoot string) error {
  remoteUrl,_ := parseRemoteDef(remote)
  fmt.Println("url", remoteUrl)
  return nil
}


func parseRemoteDef(remoteUrl string) (*url.URL, error) {
  u, err := url.Parse(remoteUrl)
	if err != nil {
		return nil, ErrInvalidRemote
	}


  fmt.Println("sch", u.Scheme, u.Path)


  return u, nil
}
