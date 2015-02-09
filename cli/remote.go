package cli

import (
	"fmt"
	"os"

	"github.com/dogestry/dogestry/remote"
)

const RemoteHelpMessage string = `  Show info about REMOTE.

  Arguments:
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
		fmt.Fprintln(os.Stderr, "Error: REMOTE not specified")
		remoteFlags.Usage()
	}

	remoteDef := remoteFlags.Arg(0)

	r, err := remote.NewRemote(remoteDef, cli.Config)
	if err != nil {
		return err
	}

	fmt.Println("remote: ", r.Desc())

	return nil
}
