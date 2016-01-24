package dockerconfig

import (
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"path/filepath"
	"strings"
)

const (
	configFileNameV1 = ".dockercfg"
	// This constant is only used for really old config files when the
	// URL wasn't saved as part of the config file and it was just
	// assumed to be this value.
	defaultIndexserver = "https://index.docker.io/v1/"
)

type v1 struct {
	ConfigReadWriter
}

func (v *v1) ConfigDir(c *ConfigFile) string {
	configDir := c.configDir
	if configDir == "" {
		configDir = getHomeDir()
	}
	return configDir
}

func (v *v1) Filename(c *ConfigFile) string {
	filename := c.filename
	if filename == "" {
		filename = configFileNameV1
	}
	return filepath.Join(v.ConfigDir(c), filename)
}

func (v *v1) LoadFromReader(r io.Reader, c *ConfigFile) error {
	b, err := ioutil.ReadAll(r)
	if err != nil {
		return err
	}

	if err := json.Unmarshal(b, &c.AuthConfigs); err != nil {
		arr := strings.Split(string(b), "\n")
		if len(arr) < 2 {
			return fmt.Errorf("The Auth config file is empty")
		}
		authConfig := AuthConfig{}
		origAuth := strings.Split(arr[0], " = ")
		if len(origAuth) != 2 {
			return fmt.Errorf("Invalid Auth config file")
		}
		authConfig.Username, authConfig.Password, err = DecodeAuth(origAuth[1])
		if err != nil {
			return err
		}
		origEmail := strings.Split(arr[1], " = ")
		if len(origEmail) != 2 {
			return fmt.Errorf("Invalid Auth config file")
		}
		authConfig.Email = origEmail[1]
		authConfig.ServerAddress = defaultIndexserver
		c.AuthConfigs[defaultIndexserver] = authConfig
	} else {
		for k, authConfig := range c.AuthConfigs {
			authConfig.Username, authConfig.Password, err = DecodeAuth(authConfig.Auth)
			if err != nil {
				return err
			}
			authConfig.Auth = ""
			authConfig.ServerAddress = k
			c.AuthConfigs[k] = authConfig
		}
	}
	return nil
}

func (v *v1) SaveToWriter(w io.Writer, c *ConfigFile) error {
	// Encode sensitive data into a new/temp struct
	tmpAuthConfigs := make(map[string]AuthConfig, len(c.AuthConfigs))
	for k, authConfig := range c.AuthConfigs {
		authCopy := authConfig
		// encode and save the authstring, while blanking out the original fields
		authCopy.Auth = EncodeAuth(&authCopy)
		authCopy.Username = ""
		authCopy.Password = ""
		authCopy.ServerAddress = ""
		tmpAuthConfigs[k] = authCopy
	}

	data, err := json.MarshalIndent(tmpAuthConfigs, "", "\t")
	if err != nil {
		return err
	}
	_, err = w.Write(data)
	return err
}
