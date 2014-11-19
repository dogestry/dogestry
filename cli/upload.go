package cli

import (
	"fmt"
)

func (cli *DogestryCli) CmdUpload(args ...string) error {
	cmd := cli.Subcmd("upload", "IMAGE_DIR IMAGE[:TAG]", "upload image saved on IMAGE_DIR into docker.")
	if err := cmd.Parse(args); err != nil {
		return nil
	}

	if len(cmd.Args()) < 2 {
		return fmt.Errorf("Error: IMAGE_DIR and IMAGE not specified")
	}

	imageDir := cmd.Arg(0)
	image := cmd.Arg(1)

	imageRoot, err := cli.WorkDirGivenBaseDir(imageDir, image)
	if err != nil {
		return err
	}

	fmt.Printf("Uploading %v as TAR file to docker host: %v\n", imageRoot, cli.DockerHost)
	if err := cli.sendTar(imageRoot); err != nil {
		return err
	}

	return nil
}
