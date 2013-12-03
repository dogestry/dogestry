package cli

import (
	"dogestry/client"
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
	remote, err := remote.NewRemote(remoteDef)
	if err != nil {
		return err
	}

	id, err := remote.ResolveImageNameToId(image)
	if err != nil {
		return err
	}

	fmt.Println("id", id)

	// TODO determine lowest missing image from docker
	remote.WalkImages(id, func(image client.Image) error {
		fmt.Println("image", image.ID)

		imageJson, err := cli.client.InspectImage(id)
		if err != nil {
			return err
		}
		fmt.Println("  json", imageJson)
		return nil
	})

	// TODO assemble tarball into imageRoot
	// TODO docker load

	return nil
}
