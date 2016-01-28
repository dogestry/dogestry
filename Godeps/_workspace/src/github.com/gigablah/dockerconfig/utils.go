package dockerconfig

import (
	"encoding/base64"
	"fmt"
	"strings"

	homedir "github.com/mitchellh/go-homedir"
)

// getHomeDir returns the user's home directory
func getHomeDir() string {
	homeDir, _ := homedir.Dir()
	return homeDir
}

// EncodeAuth creates a base64 encoded string to containing authorization information
func EncodeAuth(a *AuthConfig) string {
	authStr := a.Username + ":" + a.Password
	msg := []byte(authStr)
	encoded := make([]byte, base64.StdEncoding.EncodedLen(len(msg)))
	base64.StdEncoding.Encode(encoded, msg)
	return string(encoded)
}

// DecodeAuth decodes a base64 encoded string and returns username and password
func DecodeAuth(authStr string) (string, string, error) {
	decLen := base64.StdEncoding.DecodedLen(len(authStr))
	decoded := make([]byte, decLen)
	authByte := []byte(authStr)
	n, err := base64.StdEncoding.Decode(decoded, authByte)
	if err != nil {
		return "", "", err
	}
	if n > decLen {
		return "", "", fmt.Errorf("Something went wrong decoding auth config")
	}
	arr := strings.SplitN(string(decoded), ":", 2)
	if len(arr) != 2 {
		return "", "", fmt.Errorf("Invalid auth configuration file")
	}
	password := strings.Trim(arr[1], "\x00")
	return arr[0], password, nil
}
