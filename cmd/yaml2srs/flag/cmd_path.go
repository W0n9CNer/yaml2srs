package yaml2srs

var path string
var outputPath string

func init() {
	commandScanFolder.Flags().StringVarP(&path, "path", "p", "", "links or folder path")
	commandScanFolder.Flags().StringVarP(&outputPath, "outputPath", "o", "", "json file and srs file output path")
	commandScanFolder.MarkFlagRequired("path")
	commandScanFolder.MarkFlagRequired("outputPath")
	rootCmd.AddCommand(commandScanFolder)

	commandScanLinks.Flags().StringVarP(&path, "path", "p", "", "links or folder path")
	commandScanLinks.Flags().StringVarP(&outputPath, "outputPath", "o", "", "json file and srs file output path")
	commandScanLinks.MarkFlagRequired("path")
	commandScanLinks.MarkFlagRequired("outputPath")
	rootCmd.AddCommand(commandScanLinks)
}
