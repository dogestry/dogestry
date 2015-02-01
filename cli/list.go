package cli

import (
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/dogestry/dogestry/remote"
)

func (cli *DogestryCli) CmdList(args ...string) error {
	w := tabwriter.NewWriter(os.Stdout, 0, 4, 2, ' ', 0)
	defer w.Flush()

	remoteDef := args[0]

	r, err := remote.NewRemote(remoteDef, cli.Config)
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
