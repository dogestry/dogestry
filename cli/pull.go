package cli

import (
	"fmt"
	"github.com/dogestry/dogestry/remote"
)

func (cli *DogestryCli) CmdPull(args ...string) error {
	cmd := cli.Subcmd("pull", "REMOTE IMAGE[:TAG]", "pull IMAGE from the REMOTE and load it into docker. TAG defaults to 'latest'")
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

	fmt.Printf("Using docker endpoints for pull: %v\n", cli.PullHosts)
	fmt.Printf("S3 Connection: %v\n", r.Desc())

	fmt.Printf("Image tag: %v\n", image)

	id, err := r.ResolveImageNameToId(image)
	if err != nil {
		return err
	}

	fmt.Printf("Image '%s' resolved to ID '%s'\n", image, id.Short())

	fmt.Println("Determining which images need to be downloaded from S3...")
	downloadMap, err := cli.makeDownloadMap(r, id, imageRoot)
	if err != nil {
		return err
	}

	fmt.Println("Downloading images from S3...")
	if err := cli.downloadImages(r, downloadMap, imageRoot); err != nil {
		return err
	}

	fmt.Println("Generating repositories JSON file...")
	if err := cli.createRepositoriesJsonFile(image, imageRoot, r); err != nil {
		return err
	}

	fmt.Printf("Importing image(%s) TAR file to docker hosts: %v\n", id.Short(), cli.PullHosts)
	if err := cli.sendTar(imageRoot); err != nil {
		return err
	}

	return nil
}
