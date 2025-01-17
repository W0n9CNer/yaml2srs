package yaml2srs

import (
	"yaml2srs/internal/process"

	"github.com/spf13/cobra"
)

var commandScanLinks = &cobra.Command{
	Use:   "links",
	Short: "Specify the path of the links.txt file",
	Run: func(cmd *cobra.Command, args []string) {
		process.Start(path, outputPath, cmd.Use)
	},
}
