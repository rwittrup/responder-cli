package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

// NewRootCmd builds the responder command: one binary with
// subcommands so new testing commands can be added later.
func NewRootCmd() *cobra.Command {
	root := &cobra.Command{
		Use:   "responder",
		Short: "Test prepared911 features from a terminal",
		Long: `Responder CLI is a testing harness for prepared911 features,
starting with the features behind the portal's Responders tab.

Use "responder [command] --help" for help on a command, including
the environment variable behind every flag.`,
		SilenceErrors: true,
		SilenceUsage:  true,
	}
	root.AddCommand(newRunDemoCmd())
	root.AddCommand(newStartAppCmd())
	return root
}

// Execute runs the command, printing failures to stderr and mapping
// them to a non-zero exit code (zero on success) for scripting.
func Execute(root *cobra.Command) int {
	if err := root.Execute(); err != nil {
		fmt.Fprintf(root.ErrOrStderr(), "Error: %v\n", err)
		return 1
	}
	return 0
}
