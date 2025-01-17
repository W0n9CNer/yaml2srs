package yaml2srs

import (
	"yaml2srs/internal/process"

	"github.com/spf13/cobra"
)

var commandScanFolder = &cobra.Command{
	Use:   "folder",
	Short: "Specify the path of the folder",
	Run: func(cmd *cobra.Command, args []string) {
		process.Start(path, outputPath, cmd.Use)
	},
}
