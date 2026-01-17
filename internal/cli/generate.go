package cli

import (
	"github.com/kolah/eugene/internal/config"
	"github.com/spf13/cobra"
)

func GenerateCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "generate",
		Short: "Generate code from OpenAPI specification",
	}

	config.BindCommonFlags(cmd)
	cmd.AddCommand(NewGoCmd())

	return cmd
}
