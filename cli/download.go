package cli

import (
	"fmt"
	"github.com/dogestry/dogestry/remote"
)

func (cli *DogestryCli) CmdDownload(args ...string) error {
	cmd := cli.Subcmd("download", "REMOTE IMAGE[:TAG]", "pull IMAGE from the REMOTE and save it locally to -tempdir. TAG defaults to 'latest'")
	if err := cmd.Parse(args); err != nil {
		return nil
	}

	if len(cmd.Args()) < 2 {
		return fmt.Errorf("Error: REMOTE and IMAGE not specified")
	}

	remoteDef := cmd.Arg(0)
	image := cmd.Arg(1)

	imageRoot, err := cli.WorkDir(image)
	if err != nil {
		return err
	}
	r, err := remote.NewRemote(remoteDef, cli.Config)
	if err != nil {
		return err
	}

	fmt.Printf("S3 Connection: %v\n", r.Desc())

	fmt.Printf("Image tag: %v\n", image)

	id, err := r.ResolveImageNameToId(image)
	if err != nil {
		return err
	}

	fmt.Printf("Image '%s' resolved to ID '%s' on remote docker hosts: %v\n", image, id.Short(), cli.DockerHost)

	fmt.Println("Downloading image and its layers from S3...")
	if err := cli.pullImage(id, imageRoot, r); err != nil {
		return err
	}

	fmt.Println("Generating repositories JSON file...")
	if err := cli.createRepositoriesJsonFile(image, imageRoot, r); err != nil {
		return err
	}

	return nil
}
