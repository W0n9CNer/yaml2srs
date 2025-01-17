package outputor

import (
	"encoding/json"
	"fmt"
	"log"
	"os"

	"github.com/sagernet/sing-box/common/srs"
	"github.com/sagernet/sing-box/option"
	sing_box_json "github.com/sagernet/sing/common/json"
)

type resultRules struct {
	Version int   `json:"version"`
	Rules   []any `json:"rules"`
}

type output struct {
	RuleMap    map[string][]string
	OutputPath string
	FileName   string
	OutputJson bool
	OutputSrs  bool
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
		o.RuleMap = ruleMap
	}
}

func WithOutputPath(outputPath string) outputOption {
	return func(o *output) {
		o.OutputPath = outputPath
	}
}

func WithFileName(fileName string) outputOption {
	return func(o *output) {
		o.FileName = fileName
	}
}

func makeJsonBytes(ruleMap map[string][]string) (jsonBytes []byte, err error) {
	return json.MarshalIndent(resultRules{Version: 1, Rules: []any{ruleMap}}, "", "  ")
}

func (o *output) Output() (err error) {
	// 创建输出目录
	mkdirErr := os.MkdirAll(o.OutputPath+"/"+o.FileName+"/", 0755)
	if mkdirErr != nil {
		err = mkdirErr
		return
	}

	// 创建输出文件
	filePath := fmt.Sprintf("%s/%s/%s", o.OutputPath, o.FileName, o.FileName)

	//  格式化后的json数据
	jsonBytes, makeErr := makeJsonBytes(o.RuleMap)
	if makeErr != nil {
		err = makeErr
		return
	}

	jsonFile, createFileErr := os.Create(filePath + ".json")
	if createFileErr != nil {
		log.Printf("create json file err %v fileName %v", createFileErr.Error(), jsonFile.Name())
	} else {
		log.Println(jsonFile.Name())
		_, writeErr := jsonFile.Write(jsonBytes)
		if writeErr != nil {
			jsonFile.Close()
			os.Remove(filePath)
			log.Printf("write json file err %v fileName %v", writeErr.Error(), jsonFile.Name())
		}

	}

	srsFile, createErr := os.Create(filePath + ".srs")
	if createErr != nil {
		log.Printf("create srs file err %v fileName %v", createErr.Error(), srsFile.Name())
	} else {
		plainRuleSet, unmarshalErr := sing_box_json.UnmarshalExtended[option.PlainRuleSetCompat](jsonBytes)
		if unmarshalErr != nil {
			log.Printf("sing-box UnmarshalExtended err %v fileName %v", unmarshalErr.Error(), srsFile.Name())
			return
		} else {
			log.Println(srsFile.Name())
			writeErr := srs.Write(srsFile, plainRuleSet.Options, plainRuleSet.Version)
			if writeErr != nil {
				srsFile.Close()
				os.Remove(filePath)
				log.Printf("write srs file err %v", writeErr.Error())
				return
			}
		}
	}
	return
}
