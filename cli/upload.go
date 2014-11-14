package cli

import (
	"fmt"
)

func (cli *DogestryCli) CmdUpload(args ...string) error {
	cmd := cli.Subcmd("upload", "IMAGE_DIR", "upload image saved on IMAGE_DIR into docker.")
	if err := cmd.Parse(args); err != nil {
		return nil
	}

	if len(cmd.Args()) < 1 {
		return fmt.Errorf("Error: IMAGE_DIR not specified")
	}

	imageDir := cmd.Arg(0)

	fmt.Printf("Uploading %v as TAR file to docker host: %v\n", imageDir, cli.DockerHost)
	if err := cli.sendTar(imageDir); err != nil {
		return err
	}

	return nil
}
