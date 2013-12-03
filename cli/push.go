package cli

import (
  "archive/tar"
  "dogestry/remote"
  "fmt"
  "io"
  "io/ioutil"
  "log"
  "os"
  "path/filepath"
  "strings"
  // "github.com/bkaradzic/go-lz4"
)

func (cli *DogestryCli) CmdPush(args ...string) error {
  cmd := cli.Subcmd("push", "IMAGE[:TAG] REMOTE", "push IMAGE to the REMOTE. TAG defaults to 'latest'")
	if err := cmd.Parse(args); err != nil {
		return nil
	}

  fmt.Println("okay, pushing", args)

  if len(cmd.Args()) < 2 {
		return fmt.Errorf("Error: IMAGE and REMOTE not specified")
  }

  fmt.Println("pushing")
  image := cmd.Arg(0)
  remoteDef := cmd.Arg(1)

  imageRoot := filepath.Join(cli.TempDir(), image)

  if err := os.MkdirAll(imageRoot, os.ModeDir | 0700); err != nil {
    return err
  }

  if err := cli.prepareImage(image, imageRoot); err != nil {
    return err
  }

  fmt.Println("pushing")
  if err := remote.Push(remoteDef, image, imageRoot); err != nil {
    return err
  }

  return nil
}


func (cli *DogestryCli) prepareImage(image, root string) error {
  reader,writer := io.Pipe()
  defer writer.Close()
  defer reader.Close()

  tarball := tar.NewReader(reader)

  errch := make(chan error)

  go func() {
    // consume the tar
    for {
      log.Println("waiting")
      header, err := tarball.Next()
      if err == io.EOF {
        // end of tar archive
        log.Println("eof tar")
        break
      }
      if err != nil {
        errch <- err
        return
      }

      if err := cli.processTarEntry(root, header, tarball); err != nil {
        errch <- err
        return
      }
    }
    log.Println("tar done")

    // donno... read a bit more?
		if _, err := ioutil.ReadAll(reader); err != nil {
      errch <- err
      return
    }

    errch <- nil
  }()


  log.Println("making req")
  if err := cli.client.GetImageTarball(image, writer); err != nil {
    fmt.Println("oops", writer)
    // this should stop the tar reader
    writer.Close()
    <-errch
    return err
  }

  writer.Close()

  log.Println("req done")

  // wait for the tar reader
  if err := <-errch; err != nil {
    return err
  }
  log.Println("ok")

  return nil
}


func (cli *DogestryCli) processTarEntry(root string, header *tar.Header, tarball io.Reader) error {
  log.Printf("processing %s:\n", header.Name)

  if header.Typeflag == tar.TypeReg {
    // special case
    if filepath.Base(header.Name) == "repositories" {
      fmt.Println("repos")
    } else {
      barename := strings.TrimPrefix(header.Name, "./")

      dest := filepath.Join( root, "images", barename )
      fmt.Println(barename, "->", dest)
      if err := os.MkdirAll(filepath.Dir(dest), os.ModeDir | 0700); err != nil {
        log.Fatal(err)
      }

      fmt.Println("creating ", dest)
      destFile,err := os.Create(dest)
      if err != nil {
        log.Fatal(err)
      }
      defer destFile.Close()

      // TODO compress the layers
      //if filepath.Base(dest) == "layer.tar" {
      //}

      if wrote, err := io.Copy(destFile, tarball); err != nil {
        log.Fatalln(err)
      } else {
        log.Println("wrote", wrote)
      }
      destFile.Close()
    }
  }

  return nil
}
