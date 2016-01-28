package dockerconfig

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
)

var (
	configDir string
)

func init() {
	SetConfigDir(os.Getenv("DOCKER_CONFIG"))
}

// SetConfigDir sets the base directory the configuration file is stored in
func SetConfigDir(dir string) {
	configDir = dir
}

// AuthConfig contains authorization information for connecting to a Registry
type AuthConfig struct {
	Username      string `json:"username,omitempty"`
	Password      string `json:"password,omitempty"`
	Auth          string `json:"auth"`
	Email         string `json:"email"`
	ServerAddress string `json:"serveraddress,omitempty"`
	RegistryToken string `json:"registrytoken,omitempty"`
}

// ConfigFile structure
type ConfigFile struct {
	AuthConfigs map[string]AuthConfig `json:"auths"`
	HTTPHeaders map[string]string     `json:"HttpHeaders,omitempty"`
	PsFormat    string                `json:"psFormat,omitempty"`
	configDir   string
	filename    string
	version     int
}

// NewConfigFile initilizes an empty configuration
func NewConfigFile(fn string) *ConfigFile {
	return &ConfigFile{
		AuthConfigs: make(map[string]AuthConfig),
		HTTPHeaders: make(map[string]string),
		filename:    fn,
		version:     2,
	}
}

// ConfigReadWriter interface
type ConfigReadWriter interface {
	LoadFromReader(r io.Reader, c *ConfigFile) error
	SaveToWriter(w io.Writer, c *ConfigFile)   error
	ConfigDir(c *ConfigFile)                   string
	Filename(c *ConfigFile)                    string
}

// NewConfigReadWriter returns a ConfigReadWriter based on the version
func NewConfigReadWriter(version int) (ConfigReadWriter, error) {
	switch version {
	case 1:
		return &v1{}, nil
	case 2:
		return &v2{}, nil
	}
	return nil, fmt.Errorf("Unknown version")
}

// Load reads the config file and decodes the authorization information
func (c *ConfigFile) Load() error {
	rw, err := NewConfigReadWriter(c.version)
	if err != nil {
		return err
	}
	filename := rw.Filename(c)
	if _, err := os.Stat(filename); err != nil {
		return err
	}
	file, err := os.Open(filename)
	if err != nil {
		return err
	}
	err = rw.LoadFromReader(file, c)
	if err != nil {
		return err
	}
	defer file.Close()
	return nil
}

// Save encodes and writes out all the authorization information
func (c *ConfigFile) Save() error {
	rw, err := NewConfigReadWriter(c.version)
	if err != nil {
		return err
	}
	filename := rw.Filename(c)
	if filename == "" {
		return fmt.Errorf("Can't save config with empty filename")
	}
	if err := os.MkdirAll(filepath.Dir(filename), 0700); err != nil {
		return err
	}
	file, err := os.OpenFile(filename, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		return err
	}
	defer file.Close()
	return rw.SaveToWriter(file, c)
}

// Filename returns the current config file location
func (c *ConfigFile) Filename() string {
	rw, err := NewConfigReadWriter(c.version)
	if err != nil {
		return ""
	}
	return rw.Filename(c)
}

// ConfigDir returns the directory the config file is stored in
func (c *ConfigFile) ConfigDir() string {
	rw, err := NewConfigReadWriter(c.version)
	if err != nil {
		return ""
	}
	return rw.ConfigDir(c)
}

// Load is a convenience function that attempts to load a v2 config with v1 fallback
func Load(dir string) (*ConfigFile, error) {
	if dir == "" {
		dir = configDir
	}

	configV2 := NewConfigFile("")
	configV2.configDir = dir
	configV2.version = 2
	if err := configV2.Load(); err == nil {
		return configV2, nil
	} else if !os.IsNotExist(err) {
		return configV2, err
	}

	configV1 := NewConfigFile("")
	configV1.configDir = dir
	configV1.version = 1
	if err := configV1.Load(); err == nil {
		return configV1, nil
	} else if os.IsNotExist(err) {
		// no configs found, assume creation of new v2 config
		return configV2, nil
	} else {
		return configV1, err
	}
}
