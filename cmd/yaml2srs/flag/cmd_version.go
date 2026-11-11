package yaml2srs

import (
	"fmt"
	"runtime/debug"

	"github.com/spf13/cobra"
)

// version 是回退值。通过 go install / go build 安装时优先使用构建信息里的模块版本。
var version = "v1.1.0"

func buildVersion() string {
	if info, ok := debug.ReadBuildInfo(); ok {
		if v := info.Main.Version; v != "" && v != "(devel)" {
			return v
		}
	}
	return version
}

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
		fmt.Printf("yaml2srs version %v (sing-box %v)\n", buildVersion(), singBoxVersion())
	},
}

func init() {
	rootCmd.AddCommand(versionCmd)
}
