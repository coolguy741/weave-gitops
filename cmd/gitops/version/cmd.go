package version

import (
	"fmt"

	log "github.com/sirupsen/logrus"
	"github.com/weaveworks/go-checkpoint"
	"github.com/weaveworks/weave-gitops/pkg/flux"
	"github.com/weaveworks/weave-gitops/pkg/osys"
	"github.com/weaveworks/weave-gitops/pkg/runner"

	"github.com/spf13/cobra"
)

// The current wego version
var Version = "v0.0.0"
var GitCommit = ""
var Branch = ""
var BuildTime = ""

var Cmd = &cobra.Command{
	Use:   "version",
	Short: "Display gitops version",
	Run:   runCmd,
	PostRun: func(cmd *cobra.Command, args []string) {
		CheckVersion(CheckpointParams())
	},
}

func runCmd(cmd *cobra.Command, args []string) {
	fmt.Println("Current Version:", Version)
	fmt.Println("GitCommit:", GitCommit)
	fmt.Println("BuildTime:", BuildTime)
	fmt.Println("Branch:", Branch)

	version, err := CheckFluxVersion()
	if err != nil {
		fmt.Println(err)
		return
	}

	fmt.Println("Flux Version:", version)
}

// CheckVersion looks to see if there is a newer version of the software available
func CheckVersion(p *checkpoint.CheckParams) {
	checkResponse, err := checkpoint.Check(p)
	if err == nil && checkResponse.Outdated {
		log.Infof("gitops version %s is available; please update at %s",
			checkResponse.CurrentVersion, checkResponse.CurrentDownloadURL)
	}
}

// CheckpointParams creates the structure to pass to CheckVersion
func CheckpointParams() *checkpoint.CheckParams {
	return &checkpoint.CheckParams{
		Product: "weave-gitops",
		Version: Version,
	}
}

// CheckpointParamsWithFlags adds the object and command from the arguments list to the checkpoint parameters
func CheckpointParamsWithFlags(params *checkpoint.CheckParams, c *cobra.Command) *checkpoint.CheckParams {
	// wego uses noun verb command syntax and the parent command will have the noun and the command passed in will be the verb
	p := params
	if params == nil {
		p = CheckpointParams()
	}

	if c.HasParent() && c.Parent().Name() != "wego" {
		p.Flags = map[string]string{
			"object":  c.Parent().Name(),
			"command": c.Name(),
		}
	}

	return p
}
func CheckFluxVersion() (string, error) {
	cliRunner := &runner.CLIRunner{}
	osysClient := osys.New()
	fluxClient := flux.New(osysClient, cliRunner)

	return fluxClient.GetVersion()
}
