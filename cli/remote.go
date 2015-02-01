package cli

import (
	"fmt"
	"github.com/dogestry/dogestry/remote"
)

const RemoteHelpMessage string = `  Arguments:
    REMOTE       Name of REMOTE.

  Examples:
    dogestry remote s3://DockerBucket/Path/?region=us-east-1
    dogestry remote /path/to/images`

func (cli *DogestryCli) CmdRemote(args ...string) error {
	remoteFlags := cli.Subcmd("remote", "REMOTE", RemoteHelpMessage)
	if err := remoteFlags.Parse(args); err != nil {
		return nil
	}

	if len(remoteFlags.Args()) < 1 {
		return fmt.Errorf("Error: REMOTE not specified")
	}

	remoteDef := remoteFlags.Arg(0)

	r, err := remote.NewRemote(remoteDef, cli.Config)
	if err != nil {
		return err
	}

	fmt.Println("remote: ", r.Desc())

	return nil
}
