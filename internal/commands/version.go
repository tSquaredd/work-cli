package commands

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/tSquaredd/work-cli/internal/ui"
)

func newVersionCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print the work CLI version",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Printf("%s %s\n",
				ui.StylePrimary.Render("work"),
				ui.StyleDim.Render("v"+version),
			)
		},
	}
}
