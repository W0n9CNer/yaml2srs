package outputor

import (
	"bytes"
	"crypto/md5"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"strings"
	"testing"

	"github.com/sagernet/sing-box/common/srs"
	"github.com/sagernet/sing-box/option"
	sing_box_json "github.com/sagernet/sing/common/json"
	"gopkg.in/yaml.v3"
)

func TestOutputWritesJSONAndSRS(t *testing.T) {
	dir := t.TempDir()
	ruleMap := map[string][]string{
		"domain":         {"exact.com"},
		"domain_suffix":  {"example.com"},
		"domain_keyword": {"keyword"},
		"domain_regex":   {`^[^.]+\.star\.com$`},
		"ip_cidr":        {"10.0.0.0/8", "192.168.1.1/32"},
		"process_name":   {"ToDesk.exe"},
	}
	rawYAML := []byte("payload:\n  - DOMAIN,exact.com\n  - DOMAIN-SUFFIX,example.com\n")

	output := NewOutput(
		WithFileName("Test"),
		WithOutputPath(dir),
		WithRuleMap(ruleMap),
		WithRawFile("Test.yaml", rawYAML),
	)
	if err := output.Output(); err != nil {
		t.Fatalf("Output() error = %v", err)
	}

	jsonPath := filepath.Join(dir, "Test", "Test.json")
	content, err := os.ReadFile(jsonPath)
	if err != nil {
		t.Fatal(err)
	}
	var document struct {
		Version int                   `json:"version"`
		Rules   []map[string][]string `json:"rules"`
	}
	if err = json.Unmarshal(content, &document); err != nil {
		t.Fatal(err)
	}
	if document.Version != 1 {
		t.Fatalf("version = %d, want 1", document.Version)
	}
	// 域名类字段和 ip_cidr 在 sing-box 里同属目的地址组(OR),
	// process_name 是独立的 AND 组, 必须拆成第二条 rule。
	wantRules := []map[string][]string{
		{
			"domain":         {"exact.com"},
			"domain_suffix":  {"example.com"},
			"domain_keyword": {"keyword"},
			"domain_regex":   {`^[^.]+\.star\.com$`},
			"ip_cidr":        {"10.0.0.0/8", "192.168.1.1/32"},
		},
		{
			"process_name": {"ToDesk.exe"},
		},
	}
	if !reflect.DeepEqual(document.Rules, wantRules) {
		t.Fatalf("rules = %#v, want %#v", document.Rules, wantRules)
	}

	srsFile, err := os.Open(filepath.Join(dir, "Test", "Test.srs"))
	if err != nil {
		t.Fatal(err)
	}
	defer srsFile.Close()
	ruleSet, err := srs.Read(srsFile, false)
	if err != nil {
		t.Fatalf("srs.Read() error = %v", err)
	}
	if ruleSet.Version != 1 {
		t.Fatalf("srs version = %d, want 1", ruleSet.Version)
	}
	if len(ruleSet.Options.Rules) != 2 {
		t.Fatalf("srs rules = %d, want 2", len(ruleSet.Options.Rules))
	}
	for index, rule := range ruleSet.Options.Rules {
		if !rule.IsValid() {
			t.Fatalf("srs rule %d is invalid", index)
		}
	}

	// 未格式化的 json: 内容与格式化版本等价, 且不含多余空白。
	compactContent, err := os.ReadFile(filepath.Join(dir, "Test", "Test.min.json"))
	if err != nil {
		t.Fatal(err)
	}
	if bytes.ContainsAny(compactContent, "\n\t") {
		t.Fatalf("min json should be compact, got %q", compactContent)
	}
	var compactDocument struct {
		Version int                   `json:"version"`
		Rules   []map[string][]string `json:"rules"`
	}
	if err = json.Unmarshal(compactContent, &compactDocument); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(compactDocument, document) {
		t.Fatalf("min json = %#v, want %#v", compactDocument, document)
	}

	// md5 文件采用 md5sum 可校验的格式。
	srsBytes, err := os.ReadFile(filepath.Join(dir, "Test", "Test.srs"))
	if err != nil {
		t.Fatal(err)
	}
	md5Content, err := os.ReadFile(filepath.Join(dir, "Test", "Test.srs.md5"))
	if err != nil {
		t.Fatal(err)
	}
	wantMD5 := fmt.Sprintf("%x  Test.srs\n", md5.Sum(srsBytes))
	if string(md5Content) != wantMD5 {
		t.Fatalf("md5 file = %q, want %q", md5Content, wantMD5)
	}

	// 原始 YAML 应按字节原样写出。
	rawContent, err := os.ReadFile(filepath.Join(dir, "Test", "Test.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(rawContent, rawYAML) {
		t.Fatalf("raw yaml = %q, want %q", rawContent, rawYAML)
	}

	entries, err := os.ReadDir(filepath.Join(dir, "Test"))
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 5 {
		names := make([]string, 0, len(entries))
		for _, entry := range entries {
			names = append(names, entry.Name())
		}
		t.Fatalf("output files = %v, want 5 files", names)
	}
}

func TestOutputRejectsEmptyRules(t *testing.T) {
	dir := t.TempDir()
	output := NewOutput(WithFileName("Empty"), WithOutputPath(dir), WithRuleMap(nil))
	if err := output.Output(); !errors.Is(err, ErrNoRules) {
		t.Fatalf("Output() error = %v, want ErrNoRules", err)
	}
	if _, err := os.Stat(filepath.Join(dir, "Empty")); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("output directory should not exist, stat error = %v", err)
	}
}

func TestOutputRejectsInvalidRawFileName(t *testing.T) {
	dir := t.TempDir()
	output := NewOutput(
		WithFileName("Bad"),
		WithOutputPath(dir),
		WithRuleMap(map[string][]string{"domain": {"example.com"}}),
		WithRawFile("../escape.yaml", []byte("payload:\n  - DOMAIN,example.com\n")),
	)
	if err := output.Output(); err == nil {
		t.Fatal("Output() error = nil, want invalid raw file name error")
	}
	if _, err := os.Stat(filepath.Join(dir, "Bad", "escape.yaml")); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("raw file should not be written outside the output directory, stat error = %v", err)
	}
}

func TestOutputRejectsInvalidCIDR(t *testing.T) {
	dir := t.TempDir()
	output := NewOutput(
		WithFileName("Bad"),
		WithOutputPath(dir),
		WithRuleMap(map[string][]string{"ip_cidr": {"not-a-cidr"}}),
	)
	if err := output.Output(); err == nil {
		t.Fatal("Output() error = nil, want CIDR parse error")
	}
}

// TestValidateGeneratedDirectory 是可选的集成校验:
//
//	YAML2SRS_STAGE=/path/to/stage go test ./internal/outputor -run TestValidateGeneratedDirectory
//
// 它会重新读取生成目录里的每个 json/srs, 确认 sing-box 能解析且规则非空。
func TestValidateGeneratedDirectory(t *testing.T) {
	dir := os.Getenv("YAML2SRS_STAGE")
	if dir == "" {
		t.Skip("set YAML2SRS_STAGE to validate a generated directory")
	}

	var checked int
	err := filepath.WalkDir(dir, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() || !strings.HasSuffix(path, ".json") {
			return nil
		}
		content, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		ruleSet, err := sing_box_json.UnmarshalExtended[option.PlainRuleSetCompat](content)
		if err != nil {
			return fmt.Errorf("%v: %w", path, err)
		}
		for index, rule := range ruleSet.Options.Rules {
			if !rule.IsValid() {
				return fmt.Errorf("%v: rule %d is empty", path, index)
			}
			for _, pattern := range rule.DefaultOptions.DomainRegex {
				if _, err := regexp.Compile(pattern); err != nil {
					return fmt.Errorf("%v: rule %d bad regex %q: %w", path, index, pattern, err)
				}
			}
		}

		if strings.HasSuffix(path, ".min.json") && bytes.ContainsAny(content, "\n\t") {
			return fmt.Errorf("%v: min json is not compact", path)
		}

		// Test.json 和 Test.min.json 对应同一个 Test.srs。
		srsPath := strings.TrimSuffix(strings.TrimSuffix(path, ".min.json"), ".json") + ".srs"
		srsBytes, err := os.ReadFile(srsPath)
		if err != nil {
			return err
		}
		decoded, err := srs.Read(bytes.NewReader(srsBytes), false)
		if err != nil {
			return fmt.Errorf("%v: %w", srsPath, err)
		}
		if len(decoded.Options.Rules) != len(ruleSet.Options.Rules) {
			return fmt.Errorf("%v: srs rules = %d, json rules = %d",
				srsPath, len(decoded.Options.Rules), len(ruleSet.Options.Rules))
		}

		md5Content, err := os.ReadFile(srsPath + ".md5")
		if err != nil {
			return err
		}
		wantMD5 := fmt.Sprintf("%x  %s\n", md5.Sum(srsBytes), filepath.Base(srsPath))
		if string(md5Content) != wantMD5 {
			return fmt.Errorf("%v: md5 = %q, want %q", srsPath+".md5", md5Content, wantMD5)
		}

		if !strings.HasSuffix(path, ".min.json") {
			// 每个规则集目录还应附带生成它的原始 YAML。
			ruleName := strings.TrimSuffix(filepath.Base(path), ".json")
			var rawPath string
			for _, ext := range []string{".yaml", ".yml"} {
				candidate := filepath.Join(filepath.Dir(path), ruleName+ext)
				if _, statErr := os.Stat(candidate); statErr == nil {
					rawPath = candidate
					break
				}
			}
			if rawPath == "" {
				return fmt.Errorf("%v: raw yaml not found", path)
			}
			rawContent, err := os.ReadFile(rawPath)
			if err != nil {
				return err
			}
			var rawDocument any
			if err = yaml.Unmarshal(rawContent, &rawDocument); err != nil {
				return fmt.Errorf("%v: %w", rawPath, err)
			}
			checked++
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if checked == 0 {
		t.Fatal("no rule set validated")
	}
	t.Logf("validated %d rule sets", checked)
}

func TestMakeRuleListSplitsMixedGroups(t *testing.T) {
	rules, err := makeRuleList(map[string][]string{
		"domain_suffix":  {"example.com"},
		"process_name":   {"ToDesk.exe"},
		"process_path":   {"/usr/bin/toDesk"},
		"port":           {"80"},
		"source_ip_cidr": {"10.0.0.0/8"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(rules) != 5 {
		t.Fatalf("rules = %d, want 5: %#v", len(rules), rules)
	}
	want := []map[string]any{
		{"domain_suffix": []string{"example.com"}},
		{"source_ip_cidr": []string{"10.0.0.0/8"}},
		{"port": []uint16{80}},
		{"process_name": []string{"ToDesk.exe"}},
		{"process_path": []string{"/usr/bin/toDesk"}},
	}
	if !reflect.DeepEqual(rules, want) {
		t.Fatalf("rules = %#v, want %#v", rules, want)
	}
}

func TestOutputWritesPortRules(t *testing.T) {
	dir := t.TempDir()
	ruleMap := map[string][]string{
		"port":              {"80", "65530"},
		"port_range":        {"114:514"},
		"source_port":       {"7777"},
		"source_port_range": {"1000:2000"},
	}
	output := NewOutput(
		WithFileName("Port"),
		WithOutputPath(dir),
		WithRuleMap(ruleMap),
	)
	if err := output.Output(); err != nil {
		t.Fatalf("Output() error = %v", err)
	}

	content, err := os.ReadFile(filepath.Join(dir, "Port", "Port.json"))
	if err != nil {
		t.Fatal(err)
	}
	var document struct {
		Rules []struct {
			Port            []uint16 `json:"port"`
			PortRange       []string `json:"port_range"`
			SourcePort      []uint16 `json:"source_port"`
			SourcePortRange []string `json:"source_port_range"`
		} `json:"rules"`
	}
	if err = json.Unmarshal(content, &document); err != nil {
		t.Fatal(err)
	}
	if len(document.Rules) != 2 {
		t.Fatalf("rules = %d, want 2 (destination and source port groups)", len(document.Rules))
	}
	rule := document.Rules[0]
	if !reflect.DeepEqual(rule.Port, []uint16{80, 65530}) {
		t.Fatalf("port = %#v", rule.Port)
	}
	if !reflect.DeepEqual(rule.PortRange, []string{"114:514"}) {
		t.Fatalf("port_range = %#v", rule.PortRange)
	}
	rule = document.Rules[1]
	if !reflect.DeepEqual(rule.SourcePort, []uint16{7777}) {
		t.Fatalf("source_port = %#v", rule.SourcePort)
	}
	if !reflect.DeepEqual(rule.SourcePortRange, []string{"1000:2000"}) {
		t.Fatalf("source_port_range = %#v", rule.SourcePortRange)
	}

	// Output() 内部已用 sing-box 解码并生成 srs, 能走到这里说明端口类型被接受。
	srsFile, err := os.Open(filepath.Join(dir, "Port", "Port.srs"))
	if err != nil {
		t.Fatal(err)
	}
	defer srsFile.Close()
	if _, err = srs.Read(srsFile, false); err != nil {
		t.Fatalf("srs.Read() error = %v", err)
	}
}
