package cli

import (
  "dogestry/remote"
  "fmt"
)

func (cli *DogestryCli) CmdS3(args ...string) error {
  cmd := cli.Subcmd("s3", "REMOTE", "tests a remote")
  if err := cmd.Parse(args); err != nil {
    return nil
  }

  fmt.Println("okay, s3", args)

  if len(cmd.Args()) < 1 {
    return fmt.Errorf("Error: REMOTE not specified")
  }

  remoteDef := cmd.Arg(0)

  r, err := remote.NewRemote(remoteDef)
  if err != nil {
    return err
  }

  fmt.Println("remote: ", r.Desc())

  repoName, repoTag := remote.NormaliseImageName(cmd.Arg(1))
  id, err := r.ParseTag(repoName, repoTag)
  if err != nil {
    return err
  }

  fmt.Println("id", id)

  return nil
}
