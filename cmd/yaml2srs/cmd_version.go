package yaml2srs

import (
	"fmt"

	"github.com/spf13/cobra"
)

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print the Version number of yaml2srs",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("yaml2srs version v1.0")
	},
}

func init() {
	rootCmd.AddCommand(versionCmd)
}
