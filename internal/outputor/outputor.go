package outputor

import (
	"bytes"
	"crypto/md5"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"github.com/sagernet/sing-box/common/srs"
	"github.com/sagernet/sing-box/option"
	sing_box_json "github.com/sagernet/sing/common/json"
)

// ErrNoRules 表示规则集为空, 调用方应跳过落盘而不是生成 sing-box 无法加载的空规则。
var ErrNoRules = errors.New("empty rule set")

type resultRules struct {
	Version int              `json:"version"`
	Rules   []map[string]any `json:"rules"`
}

type output struct {
	ruleMap     map[string][]string
	outputPath  string
	fileName    string
	rawFileName string
	rawBody     []byte
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

// WithRawFile 附带该规则集对应的原始 YAML 文件。
// fileName 必须是纯文件名(不能包含路径), body 按原样写入, 不做任何改写。
func WithRawFile(fileName string, body []byte) outputOption {
	return func(o *output) {
		o.rawFileName = fileName
		o.rawBody = body
	}
}

// ruleGroups 按 sing-box headless rule 的匹配语义给字段分组:
// 同一个组里的字段在 sing-box 里是 OR 关系, 不同组之间是 AND 关系。
// Clash 规则集里的每一行是 OR, 因此不同组必须拆成多个 rule。
// process_* 在 sing-box 里分属不同的 RuleItem, 彼此也是 AND, 所以每个字段单独一组。
var ruleGroups = [][]string{
	{"domain", "domain_suffix", "domain_keyword", "domain_regex", "ip_cidr"},
	{"source_ip_cidr"},
	{"port", "port_range"},
	{"source_port", "source_port_range"},
	{"process_name"},
	{"process_path"},
	{"process_path_regex"},
}

// makeJsonBytes 生成规则集 JSON。indent 为 true 时带缩进, 否则输出紧凑格式。
func makeJsonBytes(ruleMap map[string][]string, indent bool) (jsonBytes []byte, err error) {
	rules, err := makeRuleList(ruleMap)
	if err != nil {
		return nil, err
	}
	if len(rules) == 0 {
		return nil, ErrNoRules
	}
	document := resultRules{Version: 1, Rules: rules}
	if indent {
		return json.MarshalIndent(document, "", "  ")
	}
	return json.Marshal(document)
}

// makeRuleList 把按字段聚合的规则转换成 sing-box 的 rules 数组。
// 同一个 OR 组内的字段放进同一条 rule, 不同组拆成不同的 rule,
// 这样混合类型的 Clash 规则集(例如同时有域名和进程规则)不会变成 AND。
func makeRuleList(ruleMap map[string][]string) ([]map[string]any, error) {
	rules := make([]map[string]any, 0, len(ruleGroups))
	used := make(map[string]struct{}, len(ruleMap))

	for _, fields := range ruleGroups {
		rule := make(map[string]any, len(fields))
		for _, field := range fields {
			values, ok := ruleMap[field]
			if !ok || len(values) == 0 {
				continue
			}
			converted, err := convertValues(field, values)
			if err != nil {
				return nil, err
			}
			rule[field] = converted
			used[field] = struct{}{}
		}
		if len(rule) > 0 {
			rules = append(rules, rule)
		}
	}

	// 防御性处理未归类的字段: 每个字段单独一条 rule, 保持 OR 语义且输出稳定。
	var unknown []string
	for field := range ruleMap {
		if _, ok := used[field]; !ok && len(ruleMap[field]) > 0 {
			unknown = append(unknown, field)
		}
	}
	sort.Strings(unknown)
	for _, field := range unknown {
		converted, err := convertValues(field, ruleMap[field])
		if err != nil {
			return nil, err
		}
		rules = append(rules, map[string]any{field: converted})
	}

	return rules, nil
}

// convertValues 把端口字段转成数字: sing-box 的 port/source_port 只接受 uint16。
func convertValues(field string, values []string) (any, error) {
	switch field {
	case "port", "source_port":
		ports := make([]uint16, 0, len(values))
		for _, value := range values {
			port, err := strconv.ParseUint(value, 10, 16)
			if err != nil {
				return nil, fmt.Errorf("parse %v port %q: %w", field, value, err)
			}
			ports = append(ports, uint16(port))
		}
		return ports, nil
	default:
		return values, nil
	}
}

// writeFile 先写临时文件再重命名, 避免中途失败留下半个文件
func writeFile(path string, data []byte) (err error) {
	tmpPath := path + ".tmp"
	if err = os.WriteFile(tmpPath, data, 0644); err != nil {
		_ = os.Remove(tmpPath)
		return err
	}
	if err = os.Rename(tmpPath, path); err != nil {
		_ = os.Remove(tmpPath)
		return err
	}
	return nil
}

func (o *output) Output() (err error) {
	if o.fileName == "" {
		return errors.New("empty output file name")
	}
	if len(o.ruleMap) == 0 {
		return ErrNoRules
	}

	// 原始 YAML 只保留文件名, 防止调用方传入带路径的值。
	rawFileName := o.rawFileName
	if rawFileName != "" {
		baseName := filepath.Base(rawFileName)
		if baseName != rawFileName || baseName == "." || baseName == ".." || strings.ContainsAny(baseName, `/\`) {
			return fmt.Errorf("invalid raw file name %q", rawFileName)
		}
		rawFileName = baseName
	}

	outputDir := filepath.Join(o.outputPath, o.fileName)
	filePath := filepath.Join(outputDir, o.fileName)

	// 格式化后的 json 规则集
	jsonBytes, err := makeJsonBytes(o.ruleMap, true)
	if err != nil {
		return err
	}
	// 未格式化的 json, 内容与 jsonBytes 等价
	compactJsonBytes, err := makeJsonBytes(o.ruleMap, false)
	if err != nil {
		return err
	}

	plainRuleSet, err := sing_box_json.UnmarshalExtended[option.PlainRuleSetCompat](jsonBytes)
	if err != nil {
		return fmt.Errorf("sing-box UnmarshalExtended: %w", err)
	}
	for index, rule := range plainRuleSet.Options.Rules {
		if !rule.IsValid() {
			return fmt.Errorf("sing-box invalid rule at index %d", index)
		}
	}

	// 在内存里生成 srs, 所有文件都成功后再落盘
	var buffer bytes.Buffer
	if err = srs.Write(&buffer, plainRuleSet.Options, plainRuleSet.Version); err != nil {
		return fmt.Errorf("write srs: %w", err)
	}

	// srs 的 md5 采用 md5sum 的输出格式:
	// 在该目录下执行 md5sum -c <规则名>.srs.md5 即可校验。
	digest := md5.Sum(buffer.Bytes())
	md5Line := fmt.Sprintf("%x  %s.srs\n", digest, o.fileName)

	// 所有内容都生成并校验通过后再创建目录和写文件。
	if err = os.MkdirAll(outputDir, 0755); err != nil {
		return err
	}
	if err = writeFile(filePath+".json", jsonBytes); err != nil {
		return err
	}
	if err = writeFile(filePath+".min.json", compactJsonBytes); err != nil {
		return err
	}
	if err = writeFile(filePath+".srs", buffer.Bytes()); err != nil {
		return err
	}
	if err = writeFile(filePath+".srs.md5", []byte(md5Line)); err != nil {
		return err
	}
	if rawFileName != "" {
		if err = writeFile(filepath.Join(outputDir, rawFileName), o.rawBody); err != nil {
			return err
		}
	}

	return nil
}
