package yaml2srs

import (
	"fmt"
	"runtime/debug"

	"github.com/spf13/cobra"
)

const version = "v1.1.0"

func singBoxVersion() string {
	info, ok := debug.ReadBuildInfo()
	if !ok {
		return "unknown"
	}
	for _, dep := range info.Deps {
		if dep.Path == "github.com/sagernet/sing-box" {
			return dep.Version
		}
	}
	return "unknown"
}

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print the Version number of yaml2srs",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("yaml2srs version %v (sing-box %v)\n", version, singBoxVersion())
	},
}

func init() {
	rootCmd.AddCommand(versionCmd)
}
