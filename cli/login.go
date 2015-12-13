package cli

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/docker/docker/cliconfig"
)

const LoginHelpMessage string = `  Save AWS credentials for REGISTRY.

  Arguments:
    REGISTRY       Registry name.

  Examples:
    dogestry login registry.example.com`

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
	dockerCfg, cfgErr := cliconfig.Load("")
	if cfgErr != nil {
		return cfgErr
	}

	fmt.Printf("Updating docker file %v...\n", dockerCfg.Filename())

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
	if err := dockerCfg.Save(); err != nil {
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
