package flux

import (
	"fmt"
	"os"
	"os/exec"

	"github.com/spf13/cobra"
	fluxBin "github.com/weaveworks/weave-gitops/pkg/flux"
	"github.com/weaveworks/weave-gitops/pkg/shims"
)

var Cmd = &cobra.Command{
	Use:     "flux -- [flux commands or flags]",
	Short:   "Use flux commands",
	Example: "wego flux -- install -h",
	Run:     runCmd,
}

var StatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Check the last known status of flux namespaces",
	Run:   runStatusCmd,
}

func init() {
	Cmd.AddCommand(StatusCmd)
}

// Example flux command with flags 'wego flux -- install -h'
func runCmd(cmd *cobra.Command, args []string) {
	exePath, err := fluxBin.GetFluxExePath()
	if err != nil {
		fmt.Fprintf(shims.Stderr(), "Error: %v\n", err)
		os.Exit(1)
	}

	c := exec.Command(exePath, args...)

	// run command
	if output, err := c.CombinedOutput(); err != nil {
		fmt.Fprintf(shims.Stderr(), "Error: %v\n", err)
	} else {
		fmt.Printf("Output: %s\n", output)
	}
}

func runStatusCmd(cmd *cobra.Command, args []string) {
	status, err := fluxBin.GetLatestStatusAllNamespaces()
	if err != nil {
		fmt.Fprintf(shims.Stderr(), "Error: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("Status: %s\n", status)
}
