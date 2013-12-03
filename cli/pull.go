package cli

import (
	"dogestry/remote"
	"fmt"
)

func (cli *DogestryCli) CmdPull(args ...string) error {
	cmd := cli.Subcmd("push", "IMAGE[:TAG] REMOTE", "pull IMAGE from the REMOTE and load it into docker. TAG defaults to 'latest'")
	if err := cmd.Parse(args); err != nil {
		return nil
	}

	fmt.Println("okay, pulling", args)

	if len(cmd.Args()) < 2 {
		return fmt.Errorf("Error: IMAGE and REMOTE not specified")
	}

	image := cmd.Arg(0)
	remoteDef := cmd.Arg(1)

	//imageRoot, err := cli.WorkDir(image)
	//if err != nil {
	//return err
	//}

	id, err := remote.ResolveImageName(remoteDef, image)
	if err != nil {
		return err
	}

	fmt.Println("id", id)

	//remote.WalkImages(func(image docker.Image) error {
	//})

	imageJson, err := cli.client.InspectImage(image)
	if err != nil {
		return err
	}

	fmt.Println("img", imageJson)

	return nil
}
