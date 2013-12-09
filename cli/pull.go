package cli

import (
  "dogestry/client"
  "dogestry/remote"
  "encoding/json"
  "fmt"
  "os"
  "os/exec"
  "path/filepath"
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

  imageRoot, err := cli.WorkDir(image)
  if err != nil {
    return err
  }
  r, err := remote.NewRemote(remoteDef)
  if err != nil {
    return err
  }

  id, err := remote.ResolveImageNameToId(r,image)
  if err != nil {
    return err
  }

  fmt.Println("root id", id)

  fmt.Println("preparing images")
  if err := cli.preparePullImage(id, imageRoot, r); err != nil {
    return err
  }

  fmt.Println("preparing repositories file")
  if err := prepareRepositories(image, imageRoot, r); err != nil {
    return err
  }

  if err := cli.sendTar(imageRoot); err != nil {
    return err
  }

  return nil
}

func (cli *DogestryCli) preparePullImage(fromId, imageRoot string, r remote.Remote) error {
  return r.WalkImages(fromId, func(id string, image client.Image, err error) error {
    fmt.Println("id", id, "image", image.ID)
    if err != nil {
      fmt.Println("err", err)
      return err
    }

    _, err = cli.client.InspectImage(id)
    if err == client.ErrNoSuchImage {
      return pullImage(id, filepath.Join(imageRoot, id), r)
    } else {
      return remote.BreakWalk
    }
  })
}

func pullImage(id, dst string, r remote.Remote) error {
  err := r.PullImageId(id, dst)
  if err != nil {
    return err
  }
  return processPulled(id, dst)
}


func processPulled(id, dst string) error {
  compressedLayerFile := filepath.Join(dst, "layer.tar.lz4")
  layerFile := filepath.Join(dst, "layer.tar")

  if _,err := os.Stat(compressedLayerFile); !os.IsNotExist(err) {
    fmt.Println("exists?", compressedLayerFile)
    cmd := exec.Command("./lz4", "-d", "-f", compressedLayerFile, layerFile)
    if err := cmd.Run(); err != nil {
      return err
    }

    return os.Remove(compressedLayerFile)
  }

  return nil
}


func prepareRepositories(image, imageRoot string, r remote.Remote) error {
  repoName, repoTag := remote.NormaliseImageName(image)

  id, err := r.ParseTag(repoName, repoTag)
  if err != nil {
    return err
  } else if id == "" {
    return nil
  }

  reposPath := filepath.Join(imageRoot, "repositories")
  reposFile, err := os.Create(reposPath)
  if err != nil {
    return err
  }
  defer reposFile.Close()

  repositories := map[string]Repository{}
  repositories[repoName] = Repository{}
  repositories[repoName][repoTag] = id

  return json.NewEncoder(reposFile).Encode(&repositories)
}

// stream the tarball into docker
// its easier here to use tar command, but it'd be neater to mirror Push's approach
func (cli *DogestryCli) sendTar(imageRoot string) error {
  cmd := exec.Command("/bin/tar", "cvf", "-", ".")
  cmd.Dir = imageRoot
  defer cmd.Wait()

  stdout, err := cmd.StdoutPipe()
  if err != nil {
    return err
  }

  if err := cmd.Start(); err != nil {
    return err
  }

  fmt.Println("kicking off post")
  return cli.client.PostImageTarball(stdout)
}
