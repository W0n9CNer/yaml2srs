package yaml2srs

import (
	"github.com/W0n9CNer/yaml2srs/internal/process"

	"github.com/spf13/cobra"
)

var commandScanLinks = &cobra.Command{
	Use:   process.CategoryLinks,
	Short: "Specify the path of the links.txt file",
	RunE: func(cmd *cobra.Command, args []string) error {
		return process.Start(path, outputPath, process.CategoryLinks)
	},
}
