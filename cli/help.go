package cli

import (
	"fmt"
)

func (cli *DogestryCli) CmdHelp(args ...string) error {
	if len(args) > 0 {
		method, exists := cli.getMethod(args[0])
		if !exists {
			fmt.Fprintf(cli.err, "Error: Command not found: %s\n", args[0])
		} else {
			method("--help")
			return nil
		}
	}

	help := fmt.Sprintf(
		`Usage: dogestry [OPTIONS] COMMAND [arg...]
Alternate registry and simple image storage for docker.
  Typical S3 Usage:
     export AWS_ACCESS_KEY=ABC
     export AWS_SECRET_KEY=DEF
     export DOCKER_HOST=tcp://localhost:2375
     dogestry push s3://<bucket name>/<path name>/?region=us-east-1 <image name>
     dogestry pull s3://<bucket name>/<path name>/?region=us-east-1 <image name>
     dogestry -pullhosts tcp://host-1:2375,tcp://host-2:2375 pull s3://<bucket name>/<path name>/ <image name>
  Commands:
     pull     - Pull IMAGE from S3 and load it into docker. TAG defaults to 'latest'
     push     - Push IMAGE to S3. TAG defaults to 'latest'
     remote   - Check a remote
`)
	fmt.Println(help)
	return nil
}
