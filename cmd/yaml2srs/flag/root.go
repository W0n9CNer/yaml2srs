package yaml2srs

import (
	"os"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:          "yaml2srs",
	Short:        "Convert Clash/mihomo yaml rules to sing-box json and srs rule sets",
	Long:         "将yaml格式的rules转换成sing-box可识别的srs二进制文件",
	SilenceUsage: true,
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
