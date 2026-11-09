package outputor

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/sagernet/sing-box/common/srs"
	"github.com/sagernet/sing-box/option"
	sing_box_json "github.com/sagernet/sing/common/json"
)

type resultRules struct {
	Version int   `json:"version"`
	Rules   []any `json:"rules"`
}

type output struct {
	ruleMap    map[string][]string
	outputPath string
	fileName   string
}

func NewOutput(options ...outputOption) *output {
	out := new(output)
	for _, opt := range options {
		opt(out)
	}
	return out
}

type outputOption func(o *output)

func WithRuleMap(ruleMap map[string][]string) outputOption {
	return func(o *output) {
		o.ruleMap = ruleMap
	}
}

func WithOutputPath(outputPath string) outputOption {
	return func(o *output) {
		o.outputPath = outputPath
	}
}

func WithFileName(fileName string) outputOption {
	return func(o *output) {
		o.fileName = fileName
	}
}

func makeJsonBytes(ruleMap map[string][]string) (jsonBytes []byte, err error) {
	return json.MarshalIndent(resultRules{Version: 1, Rules: []any{ruleMap}}, "", "  ")
}

// writeFile 先写临时文件再重命名, 避免中途失败留下半个文件
func writeFile(path string, data []byte) error {
	tmpPath := path + ".tmp"
	if err := os.WriteFile(tmpPath, data, 0644); err != nil {
		os.Remove(tmpPath)
		return err
	}
	return os.Rename(tmpPath, path)
}

func (o *output) Output() (err error) {
	outputDir := filepath.Join(o.outputPath, o.fileName)
	if err = os.MkdirAll(outputDir, 0755); err != nil {
		return err
	}
	filePath := filepath.Join(outputDir, o.fileName)

	// 格式化后的 json 规则集
	jsonBytes, err := makeJsonBytes(o.ruleMap)
	if err != nil {
		return err
	}
	if err = writeFile(filePath+".json", jsonBytes); err != nil {
		return err
	}

	plainRuleSet, err := sing_box_json.UnmarshalExtended[option.PlainRuleSetCompat](jsonBytes)
	if err != nil {
		return fmt.Errorf("sing-box UnmarshalExtended: %w", err)
	}

	// 在内存里生成 srs, 失败时不落盘
	var buffer bytes.Buffer
	if err = srs.Write(&buffer, plainRuleSet.Options, plainRuleSet.Version); err != nil {
		return fmt.Errorf("write srs: %w", err)
	}
	if err = writeFile(filePath+".srs", buffer.Bytes()); err != nil {
		return err
	}

	log.Println(filePath+".json", filePath+".srs")
	return nil
}
