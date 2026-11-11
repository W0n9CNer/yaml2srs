package yaml2srs

import (
	"github.com/W0n9CNer/yaml2srs/internal/process"

	"github.com/spf13/cobra"
)

var commandScanFolder = &cobra.Command{
	Use:   process.CategoryFolder,
	Short: "Specify the path of the folder",
	RunE: func(cmd *cobra.Command, args []string) error {
		return process.Start(path, outputPath, process.CategoryFolder)
	},
}
