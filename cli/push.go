package cli

import (
	"archive/tar"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/dogestry/dogestry/remote"
	"github.com/dogestry/dogestry/utils"
	docker "github.com/fsouza/go-dockerclient"
)

const PushHelpMessage string = `  Push IMAGE from docker to REMOTE.

   Arguments:
    REMOTE       Name of REMOTE.
    IMAGE[:TAG]  Name of IMAGE. TAG is optional, and defaults to 'latest'.

  Examples:
    dogestry push s3://DockerBucket/Path/?region=us-east-1 ubuntu:14.04
    dogestry push /path/to/images ubuntu`

func (cli *DogestryCli) CmdPush(args ...string) error {
	pushFlags := cli.Subcmd("push", "REMOTE IMAGE[:TAG]", PushHelpMessage)
	if err := pushFlags.Parse(args); err != nil {
		return nil
	}

	if len(pushFlags.Args()) < 2 {
		fmt.Fprintln(cli.err, "Error: IMAGE and REMOTE not specified")
		pushFlags.Usage()
		os.Exit(2)
	}

	S3URL := pushFlags.Arg(0)
	image := pushFlags.Arg(1)

	imageRoot, err := cli.WorkDir(image)
	if err != nil {
		return err
	}

	cli.Config.SetS3URL(S3URL)

	remote, err := remote.NewRemote(cli.Config)
	if err != nil {
		return err
	}

	fmt.Printf("Using docker endpoint for push: %v\n", cli.DockerHost)
	fmt.Printf("Remote: %v\n", remote.Desc())

	if err = cli.exportToFiles(image, remote, imageRoot); err != nil {
		return err
	}

	if err := remote.Push(image, imageRoot); err != nil {
		fmt.Printf(`{"Status":"error", "Message": "%v"}`+"\n", err.Error())
		return err
	}

	fmt.Println(`{"Status":"ok"}`)
	return nil
}

// There's no Set data structure in Go, so use a map to simulate one.
type set map[remote.ID]struct{}

// We don't use the value in a set, so it's always empty.
var empty struct{}

// Stream the tarball from docker and translate it into the portable repo format
// Note that its easier to handle as a stream on the way out.
func (cli *DogestryCli) exportImageToFiles(image, root string, saveIds set) error {
	fmt.Printf("Exporting image: %v to: %v\n", image, root)

	reader, writer := io.Pipe()
	defer writer.Close()
	defer reader.Close()

	tarball := tar.NewReader(reader)

	errch := make(chan error)

	go func() {
		defer close(errch)
		for {
			header, err := tarball.Next()

			if err == io.EOF {
				break
			}

			if err != nil {
				errch <- err
				return
			}

			parts := strings.Split(header.Name, "/")
			idFromFile := remote.ID(parts[0])

			if _, ok := saveIds[idFromFile]; ok {
				if err := cli.createFileFromTar(root, header, tarball); err != nil {
					errch <- err
					return
				}
			} else {
				// Drain the reader. Is this necessary?
				if _, err := io.Copy(ioutil.Discard, tarball); err != nil {
					errch <- err
					return
				}

			}
		}

		errch <- nil
	}()

	if err := cli.Client.ExportImage(docker.ExportImageOptions{image, writer}); err != nil {
		return err
	}

	// wait for the tar reader
	if err := <-errch; err != nil {
		return err
	}

	return nil
}

func (cli *DogestryCli) createFileFromTar(root string, header *tar.Header, tarball io.Reader) error {
	// only handle files (directories are implicit)
	if header.Typeflag == tar.TypeReg {
		fmt.Printf("  tar: extracting file: %s\n", header.Name)

		// special case - repositories file
		if filepath.Base(header.Name) == "repositories" {
			if err := createRepositoriesJsonFile(root, tarball); err != nil {
				return err
			}

		} else {
			barename := strings.TrimPrefix(header.Name, "./")

			dest := filepath.Join(root, "images", barename)
			if err := os.MkdirAll(filepath.Dir(dest), os.ModeDir|0700); err != nil {
				return err
			}

			destFile, err := os.Create(dest)
			if err != nil {
				return err
			}

			if wrote, err := io.Copy(destFile, tarball); err != nil {
				return err
			} else {
				fmt.Printf("  tar: file created. Size: %s\n", utils.HumanSize(wrote))
			}

			destFile.Close()
		}
	}

	return nil
}

type Repository map[string]string

func createRepositoriesJsonFile(root string, tarball io.Reader) error {
	destRoot := filepath.Join(root, "repositories")

	repositories := map[string]Repository{}
	if err := json.NewDecoder(tarball).Decode(&repositories); err != nil {
		return err
	}

	for repoName, repo := range repositories {
		for tag, id := range repo {
			dest := filepath.Join(destRoot, repoName, tag)

			if err := os.MkdirAll(filepath.Dir(dest), os.ModeDir|0700); err != nil {
				return err
			}

			if err := ioutil.WriteFile(dest, []byte(id), 0600); err != nil {
				return err
			}
		}
	}

	return nil
}

func (cli *DogestryCli) exportMetaDataToFiles(repoName string, repoTag string, id remote.ID, root string) error {
	fmt.Printf("Exporting metadata for: %v to: %v\n", repoName, root)
	dest := filepath.Join(root, "repositories", repoName, repoTag)

	if err := os.MkdirAll(filepath.Dir(dest), os.ModeDir|0700); err != nil {
		return err
	}

	if err := ioutil.WriteFile(dest, []byte(id), 0600); err != nil {
		return err
	}
	return nil
}

func (cli *DogestryCli) exportToFiles(image string, r remote.Remote, imageRoot string) error {
	imageHistory, err := cli.Client.ImageHistory(image)
	if err != nil {
		fmt.Printf("Error getting image history: %v\n", err)
		return err
	}

	fmt.Println("Checking layers on remote")

	imageID := remote.ID(imageHistory[0].ID)
	repoName, repoTag := remote.NormaliseImageName(image)

	// Check the remote to see what layers are missing. Only missing Ids will
	// need to be saved to disk when exporting the docker image.

	missingIds := make(set)

	for _, i := range imageHistory {
		id := remote.ID(i.ID)
		_, err = r.ImageMetadata(id)
		if err == nil {
			fmt.Printf("  exists   : %v\n", id)
		} else {
			fmt.Printf("  not found: %v\n", id)
			missingIds[id] = empty
		}
	}

	if len(missingIds) > 0 {
		if err := cli.exportImageToFiles(image, imageRoot, missingIds); err != nil {
			return err
		}
	}

	if err := cli.exportMetaDataToFiles(repoName, repoTag, imageID, imageRoot); err != nil {
		return err
	}

	return nil
}
