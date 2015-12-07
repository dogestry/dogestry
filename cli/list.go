package cli

import (
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/dogestry/dogestry/remote"
)

const ListHelpMessage string = `  List images on REMOTE.

  Arguments:
    REMOTE       Name of REMOTE.

  Examples:
    dogestry list s3://DockerBucket/Path/?region=us-east-1
    dogestry list /path/to/images`

func (cli *DogestryCli) CmdList(args ...string) error {
	listFlags := cli.Subcmd("list", "REMOTE", ListHelpMessage)
	if err := listFlags.Parse(args); err != nil {
		return nil
	}

	if len(listFlags.Args()) < 1 {
		fmt.Fprintln(cli.err, "Error: REMOTE not specified")
		listFlags.Usage()
		os.Exit(2)
	}

	S3URL := listFlags.Arg(0)

	w := tabwriter.NewWriter(os.Stdout, 0, 4, 2, ' ', 0)
	defer w.Flush()

	cli.Config.SetS3URL(S3URL)

	r, err := remote.NewRemote(cli.Config, nil)
	if err != nil {
		return err
	}

	images, err := r.List()
	if err != nil {
		return err
	}

	fmt.Fprintf(w, "REPOSITORY\tTAG\n")

	for _, i := range images {
		line := fmt.Sprintf("%s\t%s", i.Repository, i.Tag)
		fmt.Fprintln(w, line)
	}

	return nil
}
