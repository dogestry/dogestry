package cli

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
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

	// Perform regular pull if we are explicitly told to _not_ use dogestry server(s)
	if cli.Config.ForceLocal {
		fmt.Println("Performing regular dogestry pull (dogestry server use disabled)...")
		return cli.RegularPull(image)
	}

	// Let's try to use dogestry server(s)!
	hosts := utils.ParseHostnames(cli.PullHosts)

	haveDogestry := make([]string, 0)
	checkTimeout := time.Duration(1) * time.Second

	// Check which hosts are running the dogestry server
	for _, host := range hosts {
		if utils.DogestryServerCheck(host, cli.Config.ServerPort, checkTimeout) {
			haveDogestry = append(haveDogestry, host)
		}
	}

	// Only perform "server pull" if all hosts are running dogestry server
	if len(haveDogestry) == len(cli.PullHosts) {
		fmt.Println("Detected dogestry server on all pullhosts!")
		return cli.DogestryPull(hosts, image)
	} else {
		fmt.Println("Performing regular dogestry pull (one or more hosts is not running dogestry server)!")
		return cli.RegularPull(image)
	}
}

type HostErrTuple struct {
	Server string
	Err    error
}

func (cli *DogestryCli) DogestryPull(hosts []string, image string) error {
	// Generate our auth header
	authHeader, headerErr := cli.GenerateAuthHeader()
	if headerErr != nil {
		return headerErr
	}

	tupleChan := make(chan *HostErrTuple, 1)

	for _, host := range hosts {
		fmt.Printf("Launching goroutine for pulling image on %v...\n", host)

		fullURL := fmt.Sprintf("http://%v:%v/1.19/images/create?fromImage=%v", host,
			cli.Config.ServerPort, url.QueryEscape(image))

		// POST and evaluate JSON stream updates
		go cli.PerformDogestryPull(fullURL, host, authHeader, tupleChan)
	}

	errorMessage := ""
	finished := make([]string, 0)

	// Listen for updates from all goroutines
	for hostStatus := range tupleChan {
		if hostStatus.Err != nil {
			// Bail if we run into an error on any host
			errorMessage = fmt.Sprintf("%v: %v; ", hostStatus.Server, hostStatus.Err.Error())
			break
		} else {
			// Received update, no error == goroutine has finished
			finished = append(finished, hostStatus.Server)
		}

		// All goroutines have finished
		if len(finished) == len(hosts) {
			break
		}
	}

	close(tupleChan)

	if errorMessage != "" {
		return fmt.Errorf("Ran into one or more errors: %v", errorMessage)
	}

	return nil
}

func (cli *DogestryCli) PerformDogestryPull(fullURL, host, authHeader string, tupleChan chan *HostErrTuple) {
	// Request dogestry server to pull image
	req, requestErr := http.NewRequest("POST", fullURL, nil)
	if requestErr != nil {
		tupleChan <- &HostErrTuple{
			Server: host,
			Err:    fmt.Errorf("Error when generating new POST request: %v", requestErr),
		}
		return
	}

	req.Header.Set("X-Registry-Auth", authHeader)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}

	resp, httpErr := client.Do(req)
	if httpErr != nil {
		tupleChan <- &HostErrTuple{
			Server: host,
			Err:    fmt.Errorf("Error when POST'ing to remote dogestry server: %v", httpErr),
		}
		return
	}
	defer resp.Body.Close()

	// Evaluate and display streamed updates from Dogestry server
	cli.StreamUpdates(host, resp.Body, tupleChan)
}

func (cli *DogestryCli) StreamUpdates(host string, body io.ReadCloser, tupleChan chan *HostErrTuple) {
	d := json.NewDecoder(body)

	var serverError error

	for {
		var statusUpdate map[string]interface{}

		if err := d.Decode(&statusUpdate); err != nil {
			// Not sure if we ever hit this state; keeping just in case.
			if err == io.EOF {
				fmt.Printf("Hmmm - reached EOF for host %v\n", host)
				break
			} else if err == io.ErrUnexpectedEOF {
				fmt.Printf("[ERROR] %v: %v\n", host, err)
				serverError = fmt.Errorf("Server disappeared: %v", err)
				break
			}
		}

		if _, ok := statusUpdate["error"]; ok {
			fmt.Printf("[ERROR] %v: %v\n", host, statusUpdate["error"].(string))
			serverError = fmt.Errorf("Error on host %v: %v", host, statusUpdate["error"].(string))
			break
		} else if _, ok := statusUpdate["status"]; ok {
			statusMessage := statusUpdate["status"].(string)

			if statusMessage == "Done" {
				fmt.Printf("[DONE] %v: Pull finished successfully\n", host)
				break
			} else {
				fmt.Printf("[UPDATE] %v: %v\n", host, statusMessage)
			}
		}
	}

	// Stream finished, send update
	tupleChan <- &HostErrTuple{
		Server: host,
		Err:    serverError,
	}
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
