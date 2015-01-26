package cli

import (
	"encoding/json"
	"github.com/dogestry/dogestry/remote"
	"github.com/dogestry/dogestry/utils"
	docker "github.com/fsouza/go-dockerclient"

	"archive/tar"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
)

func (cli *DogestryCli) CmdPush(args ...string) error {
	cmd := cli.Subcmd("push", "REMOTE IMAGE[:TAG]", "push IMAGE to the REMOTE. TAG defaults to 'latest'")
	if err := cmd.Parse(args); err != nil {
		return nil
	}

	if len(cmd.Args()) < 2 {
		return fmt.Errorf("Error: IMAGE and REMOTE not specified")
	}

	remoteDef := cmd.Arg(0)
	image := cmd.Arg(1)

	imageRoot, err := cli.WorkDir(image)
	if err != nil {
		return err
	}

	remote, err := remote.NewRemote(remoteDef, cli.Config)
	if err != nil {
		return err
	}

	fmt.Printf("Remote: %v\n", remote.Desc())

	fmt.Println("Exporting files")
	if err = cli.exportToFiles(image, remote, imageRoot); err != nil {
		return err
	}

	fmt.Println("Pushing image to remote")
	if err := remote.Push(image, imageRoot); err != nil {
		return err
	}

	return nil
}

// Stream the tarball from docker and translate it into the portable repo format
// Note that its easier to handle as a stream on the way out.
func (cli *DogestryCli) exportImageToFiles(image, root string) error {
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

			if err := cli.createFileFromTar(root, header, tarball); err != nil {
				errch <- err
				return
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
	}

	imageID := remote.ID(imageHistory[0].ID)
	repoName, repoTag := remote.NormaliseImageName(image)

	// In theory, we could just check the "top layer" and assume that, if it exists, then
	// that means all the parent layers are present. Instead, this checks all image layers,
	// which is a little safer, but slower.
	//
	// As soon as one check fails, we can break out of the loop, since that means we will have
	// to export the entire image.
	for _, i := range imageHistory {
		id := remote.ID(i.ID)
		fmt.Printf("  checking id: %v\n", id)
		_, err = r.ImageMetadata(id)
		if err != nil {
			break
		}
	}

	if err == nil {
		if err := cli.exportMetaDataToFiles(repoName, repoTag, imageID, imageRoot); err != nil {
			return err
		}
	} else {
		if err := cli.exportImageToFiles(image, imageRoot); err != nil {
			return err
		}
	}
	return nil
}
