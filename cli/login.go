package cli

import (
	"bufio"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"os/user"
	"strings"
)

const LoginHelpMessage string = `  Save AWS credentials for REGISTRY.

  Arguments:
    REGISTRY       Registry name.

  Examples:
    dogestry login registry.example.com`

type ConfigFile struct {
	AuthConfigs map[string]AuthEntry `json:"auths"`
}

type AuthEntry struct {
	Auth  string `json:"auth"`
	Email string `json:"email"`
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

	// Try to locate a .dockercfg
	dockerFile, cfgErr := LocateDockerCfg()
	if cfgErr != nil {
		return cfgErr
	}

	fmt.Printf("Updating docker file %v...\n", dockerFile)

	// Get input
	loginInfo, inputErr := GetLoginInput()
	if inputErr != nil {
		return inputErr
	}

	// Update .dockercfg
	if err := UpdateDockerCfg(url, dockerFile, loginInfo); err != nil {
		return err
	}

	return nil
}

func UpdateDockerCfg(url, dockerFile string, loginInfo map[string]string) error {
	// Read the contents of the file
	authConfig := make(map[string]*AuthEntry)

	if _, err := os.Stat(dockerFile); err == nil {
		fileContents, readErr := ioutil.ReadFile(dockerFile)
		if readErr != nil {
			return fmt.Errorf("Unable to read existing docker config: %v", readErr)
		}

		if err := json.Unmarshal(fileContents, &authConfig); err != nil {
			return fmt.Errorf("Unable to parse existing docker config: %v", err)
		}
	}

	// Encode data
	authString := loginInfo["AWS_ACCESS_KEY"] + ":" + loginInfo["AWS_SECRET_KEY"]

	encoded := base64.StdEncoding.EncodeToString([]byte(authString))

	// Update authConfig
	if _, ok := authConfig[url]; !ok {
		authConfig[url] = &AuthEntry{}
	}

	authConfig[url].Auth = encoded
	authConfig[url].Email = loginInfo["S3_URL"]

	jsonData, marshalErr := json.MarshalIndent(authConfig, "", "\t")
	if marshalErr != nil {
		return fmt.Errorf("Unable to generate new JSON .dockercfg: %v", marshalErr)
	}

	// Save it all
	if err := ioutil.WriteFile(dockerFile, jsonData, 0600); err != nil {
		return fmt.Errorf("Unable to write new .dockercfg: %v", err)
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

func LocateDockerCfg() (string, error) {
	usr, err := user.Current()
	if err != nil {
		return "", err
	}

	// Check if .config dir exists
	s, err := os.Stat(usr.HomeDir + "/.docker")
	if err != nil {
		// File/dir does not exist, fall back
		return usr.HomeDir + "/.dockercfg", nil
	}

	// Exists, but is it a dir?
	if s.IsDir() {
		return usr.HomeDir + "/.docker/config", nil
	}

	// Not a dir, fall back
	return usr.HomeDir + "/.dockercfg", nil
}
