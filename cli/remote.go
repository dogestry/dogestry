package cli

import (
	"dogestry/remote"
	"fmt"
)

func (cli *DogestryCli) CmdRemote(args ...string) error {
	cmd := cli.Subcmd("remote", "REMOTE", "describes a remote")
	if err := cmd.Parse(args); err != nil {
		return nil
	}

	fmt.Println("okay, remote", args)

	if len(cmd.Args()) < 1 {
		return fmt.Errorf("Error: REMOTE not specified")
	}

	remoteDef := cmd.Arg(0)

	r, err := remote.NewRemote(remoteDef)
	if err != nil {
		return err
	}

	fmt.Println("remote: ", r.Desc())

	return nil
}
