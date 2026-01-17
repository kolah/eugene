package cli

import "github.com/spf13/cobra"

func RootCmd() *cobra.Command {
	root := &cobra.Command{
		Use:     "eugene",
		Short:   "Eugene - OpenAPI INterface Kit - oink! 🐷",
		Version: "1.0.0",

		RunE: func(cmd *cobra.Command, args []string) error {
			return cmd.Help()
		},
	}

	root.AddCommand(GenerateCommand())

	return root
}
