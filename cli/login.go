package cli

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/docker/docker/cliconfig"
	homedir "github.com/mitchellh/go-homedir"
)

const LoginHelpMessage string = `  Save AWS credentials for REGISTRY.

  Arguments:
    REGISTRY       Registry name.

  Examples:
    dogestry login registry.example.com`

type CompatConfigFile struct {
	*cliconfig.ConfigFile
	filename string
}

func (cli *DogestryCli) CmdLogin(args ...string) error {
	loginFlags := cli.Subcmd("login", "REMOTE", LoginHelpMessage)
	if err := loginFlags.Parse(args); err != nil {
		return nil
	}

	if len(loginFlags.Args()) < 1 {
		fmt.Fprintln(cli.err, "Error: REMOTE not specified")
		loginFlags.Usage()
		os.Exit(2)
	}

	url := loginFlags.Arg(0)

	// Try to locate a docker config
	dockerCfg, cfgErr := CompatLoad()
	if cfgErr != nil {
		return cfgErr
	}

	fmt.Printf("Updating docker file %v...\n", dockerCfg.filename)

	// Get input
	loginInfo, inputErr := GetLoginInput()
	if inputErr != nil {
		return inputErr
	}

	authconfig, ok := dockerCfg.AuthConfigs[url]
	if !ok {
		authconfig = cliconfig.AuthConfig{}
	}
	authconfig.Username = loginInfo["AWS_ACCESS_KEY"]
	authconfig.Password = loginInfo["AWS_SECRET_KEY"]
	authconfig.Email = loginInfo["S3_URL"]
	authconfig.ServerAddress = url
	dockerCfg.AuthConfigs[url] = authconfig

	// Update docker config
	if err := dockerCfg.CompatSave(); err != nil {
		return err
	}

	return nil
}

func GetLoginInput() (map[string]string, error) {
	loginInfoKeys := []string{"AWS_ACCESS_KEY", "AWS_SECRET_KEY", "S3_URL"}
	loginInfo := make(map[string]string, 0)

	reader := bufio.NewReader(os.Stdin)

	for _, k := range loginInfoKeys {
		fmt.Printf("%v: ", k)
		value, _ := reader.ReadString('\n')
		value = strings.TrimSpace(value)

		if value == "" {
			return nil, fmt.Errorf("'%v' cannot be blank!", k)
		}

		loginInfo[k] = value
	}

	return loginInfo, nil
}

func (configFile *CompatConfigFile) CompatSave() error {
	if strings.HasSuffix(configFile.filename, ".dockercfg") {
		if err := os.MkdirAll(filepath.Dir(configFile.filename), 0700); err != nil {
			return err
		}
		f, err := os.OpenFile(configFile.filename, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0600)
		if err != nil {
			return err
		}
		defer f.Close()
		return configFile.LegacySaveToWriter(f)
	}

	return configFile.Save()
}

func (configFile *CompatConfigFile) LegacySaveToWriter(writer io.Writer) error {
	// Encode sensitive data into a new/temp struct
	tmpAuthConfigs := make(map[string]cliconfig.AuthConfig, len(configFile.AuthConfigs))
	for k, authConfig := range configFile.AuthConfigs {
		authCopy := authConfig
		// encode and save the authstring, while blanking out the original fields
		authCopy.Auth = cliconfig.EncodeAuth(&authCopy)
		authCopy.Username = ""
		authCopy.Password = ""
		authCopy.ServerAddress = ""
		tmpAuthConfigs[k] = authCopy
	}

	saveAuthConfigs := configFile.AuthConfigs
	configFile.AuthConfigs = tmpAuthConfigs
	defer func() { configFile.AuthConfigs = saveAuthConfigs }()

	data, err := json.MarshalIndent(configFile.AuthConfigs, "", "\t")
	if err != nil {
		return err
	}
	_, err = writer.Write(data)
	return err
}

func CompatLoad() (*CompatConfigFile, error) {
	dockerCfg := cliconfig.NewConfigFile(filepath.Join(cliconfig.ConfigDir(), cliconfig.ConfigFileName))
	configFile := CompatConfigFile{
		dockerCfg,
		dockerCfg.Filename(),
	}

	// Try .docker/config.json first
	if _, err := os.Stat(configFile.filename); err == nil {
		file, err := os.Open(configFile.filename)
		if err != nil {
			return &configFile, err
		}
		defer file.Close()
		err = configFile.LoadFromReader(file)
		return &configFile, err
	} else if !os.IsNotExist(err) {
		return &configFile, err
	}

	// Try the old .dockercfg
	homeDir, _ := homedir.Dir()
	configFile.filename = filepath.Join(homeDir, ".dockercfg")
	if _, err := os.Stat(configFile.filename); err != nil {
		return &configFile, nil //missing file is not an error
	}
	file, err := os.Open(configFile.filename)
	if err != nil {
		return &configFile, err
	}
	defer file.Close()
	err = configFile.LegacyLoadFromReader(file)
	if err != nil {
		return &configFile, err
	}

	return &configFile, nil
}
