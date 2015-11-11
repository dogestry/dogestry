package cli

import (
	"fmt"
)

const HelpMessage string = `Usage: dogestry [OPTIONS] COMMAND [arg...]

  Commands:
     help          Print help message. Use help COMMAND for more specific help
     list          List images on remote
     pull          Pull IMAGE from remote and load it into docker
     push          Push IMAGE from docker to remote
     remote        Show info about remote
     version       Print version
     login         Add your AWS credentials to your .dockercfg (similar to 'docker login')

  Options:
     -config       Path to optional config file
     -pullhosts    A comma-separated list of docker hosts where the image will be pulled
     -lockfile     Path to optional lock file to use, prevents parallel execution
     -server       Run dogestry in server mode
     -address      What address to bind to for dogestry server mode (default: 0.0.0.0)
     -port         What port to use for dogestry server (default: 22375)
     -force-local  Do *not* attempt to utilize remote Dogestry servers (default: false)

  Typical S3 Usage:
     dogestry push s3://<bucket name>/<path name>/?region=us-east-1 <image name>
     dogestry pull s3://<bucket name>/<path name>/?region=us-west-1 <image name>
     dogestry -pullhosts tcp://host-1:2375,tcp://host-2:2375 pull s3://<bucket name>/<path name>/ <image name>
`

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

	fmt.Println(HelpMessage)
	return nil
}
