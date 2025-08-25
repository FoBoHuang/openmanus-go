package commands

import "github.com/spf13/cobra"

// NewRootCommand wires all subcommands
func NewRootCommand() *cobra.Command {
	root := &cobra.Command{Use: "openmanus"}
	root.AddCommand(NewRunCommand())
	root.AddCommand(NewMCPCommand())
	return root
}
