package yaml2srs

import (
	"github.com/spf13/cobra"
)

var path string
var outputPath string

func init() {
	commandScanFolder.Flags().StringVarP(&path, "path", "p", "", "links or folder path")
	commandScanFolder.Flags().StringVarP(&outputPath, "outputPath", "o", "", "rule set output directory")
	cobra.CheckErr(commandScanFolder.MarkFlagRequired("path"))
	cobra.CheckErr(commandScanFolder.MarkFlagRequired("outputPath"))
	rootCmd.AddCommand(commandScanFolder)

	commandScanLinks.Flags().StringVarP(&path, "path", "p", "", "links or folder path")
	commandScanLinks.Flags().StringVarP(&outputPath, "outputPath", "o", "", "rule set output directory")
	cobra.CheckErr(commandScanLinks.MarkFlagRequired("path"))
	cobra.CheckErr(commandScanLinks.MarkFlagRequired("outputPath"))
	rootCmd.AddCommand(commandScanLinks)
}
