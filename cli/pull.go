package cli

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"time"

	"github.com/dogestry/dogestry/config"
	"github.com/dogestry/dogestry/remote"
	"github.com/dogestry/dogestry/utils"
)

const PullHelpMessage string = `  Pull IMAGE from REMOTE and load it into docker.

  Arguments:
    REMOTE       Name of REMOTE.
    IMAGE[:TAG]  Name of IMAGE. TAG is optional, and defaults to 'latest'.

  Examples:
    dogestry -pullhosts tcp://host-1:2375 pull s3://DockerBucket/Path/ ubuntu:14.04
    dogestry pull /path/to/images ubuntu`

func (cli *DogestryCli) CmdPull(args ...string) error {
	pullFlags := cli.Subcmd("pull", "REMOTE IMAGE[:TAG]", PullHelpMessage)

	// Don't return error here, this part is only relevant for CLI
	if err := pullFlags.Parse(args); err != nil {
		return nil
	}

	if len(pullFlags.Args()) < 2 {
		return errors.New("Error: REMOTE and IMAGE not specified")
	}

	S3URL := pullFlags.Arg(0)
	image := pullFlags.Arg(1)

	cli.Config.SetS3URL(S3URL)

	// Extract hostname from pullhosts args
	hosts := utils.ParseHostnames(cli.PullHosts)

	haveDogestry := make([]string, 0)
	checkTimeout := time.Duration(1) * time.Second

	// Check which hosts are running the dogestry server
	for _, host := range hosts {
		if utils.DogestryServerCheck(host, cli.Config.ServerPort, checkTimeout) {
			haveDogestry = append(haveDogestry, host)
		}
	}

	if len(haveDogestry) == len(cli.PullHosts) {
		fmt.Println("Detected dogestry server on all pullhosts!")
		return cli.DogestryPull(hosts, image)
	} else {
		fmt.Println("Performing regular dogestry pull (this may take a while)!")
		return cli.RegularPull(image)
	}
}

func (cli *DogestryCli) DogestryPull(hosts []string, image string) error {
	// Generate our auth header
	authHeader, headerErr := cli.GenerateAuthHeader()
	if headerErr != nil {
		return headerErr
	}

	type hostErrTuple struct {
		server string
		err    error
	}

	tupleChan := make(chan hostErrTuple)

	for _, host := range hosts {
		fmt.Printf("Starting image pull via dogestry server on host %v...\n", host)

		// Craft url
		fullURL := fmt.Sprintf("http://%v:%v/9001/images/create?fromImage=%v", host,
			cli.Config.ServerPort, url.QueryEscape(image))

		go func(host string, header string, tupleChan chan hostErrTuple) {
			// Request dogestry server to pull image
			req, requestErr := http.NewRequest("POST", fullURL, nil)
			if requestErr != nil {
				tupleChan <- hostErrTuple{
					server: host,
					err:    fmt.Errorf("Error when generating new POST request: %v", requestErr),
				}
				return
			}

			req.Header.Set("X-Registry-Auth", authHeader)
			req.Header.Set("Content-Type", "application/json")

			client := &http.Client{}

			resp, httpErr := client.Do(req)
			if httpErr != nil {

				tupleChan <- hostErrTuple{
					server: host,
					err:    fmt.Errorf("Error when POST'ing to remote dogestry server: %v", httpErr),
				}
				return
			}
			defer resp.Body.Close()

			body, readErr := ioutil.ReadAll(resp.Body)
			if readErr != nil {
				tupleChan <- hostErrTuple{
					server: host,
					err:    fmt.Errorf("Error reading dogestry server's response: %v", readErr),
				}
				return
			}

			fmt.Printf("Go routine for host %v got this response: %v\n", host, string(body))

			// Unmarshal and check JSON

			// All is well
			tupleChan <- hostErrTuple{"", nil}
		}(host, authHeader, tupleChan)
	}

	errorMessage := ""

	// Listen for updates from all goroutines
	for range hosts {
		hostStatus := <-tupleChan

		if hostStatus.err != nil {
			// Combine all errors into a single message
			errorMessage = errorMessage + fmt.Sprintf("%v: %v; ", hostStatus.server, hostStatus.err.Error())
		}
	}

	close(tupleChan)

	if errorMessage != "" {
		return fmt.Errorf("Ran into one or more errors: %v", errorMessage)
	}

	return nil
}

func (cli *DogestryCli) GenerateAuthHeader() (string, error) {
	authHeader := &config.AuthConfig{
		Username: cli.Config.AWS.AccessKeyID,
		Password: cli.Config.AWS.SecretAccessKey,
		Email:    cli.Config.AWS.S3URL.String(),
	}

	data, err := json.Marshal(authHeader)
	if err != nil {
		return "", err
	}

	encData := base64.StdEncoding.EncodeToString(data)

	return encData, nil
}

func (cli *DogestryCli) RegularPull(image string) error {
	imageRoot, err := cli.WorkDir(image)
	if err != nil {
		return err
	}

	r, err := remote.NewRemote(cli.Config)
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
