package process

import (
	"errors"
	"fmt"
	"log"
	"net/netip"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"gopkg.in/yaml.v3"
)

// errNoValidRule 表示源文件里没有任何可以写入规则集的条目。
// 这类文件会被跳过, 但不会让整次生成失败。
var errNoValidRule = errors.New("no valid rule found")

// KEY: https://wiki.metacubex.one/config/rules/
// VALUE: https://sing-box.sagernet.org/configuration/route/rule/
// Clash 规则类型与 sing-box 规则集的对应关系。
var mapDictMap = map[string]string{
	"DOMAIN":         "domain",
	"DOMAIN-SUFFIX":  "domain_suffix",
	"DOMAIN-KEYWORD": "domain_keyword",
	"DOMAIN-REGEX":   "domain_regex",
	// "DOMAIN-WILDCARD": "", // 不映射: 通配语义与 sing-box 的 domain_regex 不完全一致
	// sing-box 的 rule-set 支持 process_name/process_path/process_path_regex,
	// 因此把 Clash 的进程规则也带上, 避免 RemoteDesktop 这类规则集变成空文件。
	"PROCESS-NAME":          "process_name",
	"PROCESS-PATH":          "process_path",
	"PROCESS-PATH-REGEX":    "process_path_regex",
	"PROCESS-PATH-WILDCARD": "process_path_wildcard",
	"IP-CIDR":               "ip_cidr",
	"IP-CIDR6":              "ip_cidr",
	"IP-SUFFIX":             "ip_cidr",
	"SRC-IP-CIDR":           "source_ip_cidr",
	"SRC-IP-SUFFIX":         "source_ip_cidr",
	"DST-PORT":              "port",
	"SRC-PORT":              "source_port",
	// "IP-ASN":         "", // sing-box 1.11 起规则集不再支持 ip_asn
	// "GEOIP":     "geoip",
	// "SRC-GEOIP": "source_geoip",
	// "SRC-IP-ASN":     "",
	// "IN-PORT":        "",
	// "IN-TYPE":        "",
	// "IN-USER":        "",
	// "IN-NAME":        "",
	// "PROCESS-NAME-WILDCARD": "",
	// "PROCESS-NAME-REGEX":    "",
	// "UID":                   "user_id",
	// "NETWORK":               "network",
	// "DSCP":                  "",
}

type parser struct {
	ruleMap     map[string][]string
	seen        map[string]map[string]struct{}
	unsupported map[string]int
	invalid     map[string]int
}

func newParser() *parser {
	return &parser{
		ruleMap:     make(map[string][]string),
		seen:        make(map[string]map[string]struct{}),
		unsupported: make(map[string]int),
		invalid:     make(map[string]int),
	}
}

func (p *parser) appendRule(pattern, address string) {
	if pattern == "" || address == "" {
		return
	}
	if _, ok := p.seen[pattern]; !ok {
		p.seen[pattern] = make(map[string]struct{})
	}
	if _, ok := p.seen[pattern][address]; ok {
		return
	}
	p.seen[pattern][address] = struct{}{}
	p.ruleMap[pattern] = append(p.ruleMap[pattern], address)
}

// add 把一条 Clash 规则(payload 中的一行)转换成 sing-box 规则项。
func (p *parser) add(ruleInfo string) {
	ruleInfo = strings.TrimSpace(ruleInfo)
	if ruleInfo == "" {
		return
	}

	// https://wiki.metacubex.one/config/rule-providers/content/
	// payload:
	// - DOMAIN-SUFFIX,google.com
	// - DOMAIN-KEYWORD,google
	// - DOMAIN,ad.com
	// - SRC-IP-CIDR,192.168.1.201/32
	// - IP-CIDR,127.0.0.0/8
	// - GEOIP,CN
	// - DST-PORT,80
	// - SRC-PORT,7777

	// https://wiki.metacubex.one/handbook/syntax/#_8
	// fake-ip-filter:
	// - ".lan"
	// - "xbox.*.microsoft.com"
	// - "+.xboxlive.com"
	// - localhost.ptlogin2.qq.com

	ruleType, rest, hasComma := strings.Cut(ruleInfo, ",")
	if hasComma {
		ruleType = strings.ToUpper(strings.TrimSpace(ruleType))
		kind, ok := mapDictMap[ruleType]
		if !ok {
			p.unsupported[ruleType]++
			return
		}

		address := strings.TrimSpace(rest)
		switch {
		case kind == "domain_regex":
			// 正则本身可能包含逗号, 不能按逗号截断。
		case strings.HasPrefix(kind, "process_"):
			// 进程路径在极端情况下也可能包含逗号, 保留整段。
		case kind == "port", kind == "source_port":
			// mihomo 的端口列表用 , 或 / 分隔, 不能按逗号截断。
		default:
			// DOMAIN-SUFFIX,example.com,no-resolve 之类的附加字段直接忽略。
			if index := strings.IndexByte(address, ','); index >= 0 {
				address = strings.TrimSpace(address[:index])
			}
		}
		p.addTyped(kind, address)
		return
	}

	p.addBare(ruleInfo)
}

func (p *parser) addTyped(kind, address string) {
	address = strings.TrimSpace(strings.Trim(address, `"'`))
	if address == "" {
		p.invalid[kind]++
		return
	}

	switch kind {
	case "domain", "domain_suffix", "domain_keyword":
		normalizedKind, normalizedAddress, ok := normalizeDomain(kind, address)
		if !ok {
			p.invalid[kind]++
			return
		}
		p.appendRule(normalizedKind, normalizedAddress)
	case "domain_regex":
		if !validRegex(address) {
			p.invalid[kind]++
			return
		}
		p.appendRule(kind, address)
	case "ip_cidr", "source_ip_cidr":
		canonical, ok := canonicalIPOrCIDR(address)
		if !ok {
			p.invalid[kind]++
			return
		}
		p.appendRule(kind, canonical)
	case "port", "source_port":
		p.addPorts(kind, address)
	case "process_path_wildcard":
		pattern := pathWildcardToRegexp(address)
		if !validRegex(pattern) {
			p.invalid[kind]++
			return
		}
		p.appendRule("process_path_regex", pattern)
	case "process_name", "process_path", "process_path_regex":
		if kind == "process_path_regex" && !validRegex(address) {
			p.invalid[kind]++
			return
		}
		p.appendRule(kind, address)
	default:
		p.unsupported[kind]++
	}
}

// addPorts 解析 mihomo 的端口写法, 例如 "80"、"114-514/810-1919,65530"。
// 单个端口写入 port/source_port, 区间写入 port_range/source_port_range。
func (p *parser) addPorts(kind, address string) {
	source := kind == "source_port"
	portKind, rangeKind := "port", "port_range"
	if source {
		portKind, rangeKind = "source_port", "source_port_range"
	}

	added := 0
	failed := 0
	for _, token := range strings.FieldsFunc(address, func(r rune) bool {
		return r == '/' || r == ','
	}) {
		token = strings.TrimSpace(token)
		if token == "" {
			continue
		}
		separator := "-"
		if !strings.Contains(token, separator) {
			separator = ":"
		}
		if strings.Contains(token, separator) {
			startText, endText, _ := strings.Cut(token, separator)
			start, startOK := parsePort(startText)
			end, endOK := parsePort(endText)
			if !startOK || !endOK || start > end {
				failed++
				continue
			}
			p.appendRule(rangeKind, fmt.Sprintf("%d:%d", start, end))
			added++
			continue
		}
		port, ok := parsePort(token)
		if !ok {
			failed++
			continue
		}
		p.appendRule(portKind, strconv.Itoa(int(port)))
		added++
	}
	if added == 0 && failed == 0 {
		p.invalid[kind]++
	}
	if failed > 0 {
		p.invalid[kind] += failed
	}
}

func parsePort(value string) (uint64, bool) {
	value = strings.TrimSpace(value)
	if value == "" {
		return 0, false
	}
	port, err := strconv.ParseUint(value, 10, 16)
	if err != nil || port == 0 {
		return 0, false
	}
	return port, true
}

// addBare 处理没有规则类型前缀的行:
//   - 1.2.3.4 / 1.2.3.0/24      -> ip_cidr
//   - .example.com              -> domain_suffix (不含 example.com 本身)
//   - +.example.com             -> domain_suffix (含 example.com 本身)
//   - *.example.com             -> domain_regex (只匹配一级)
//   - example.com               -> domain (精确匹配)
func (p *parser) addBare(ruleInfo string) {
	kind, address, ok := normalizeDomain("domain", ruleInfo)
	if !ok {
		p.invalid["domain"]++
		return
	}
	p.appendRule(kind, address)
}

func normalizeDomain(kind, address string) (string, string, bool) {
	address = strings.TrimSpace(strings.Trim(address, `"'`))
	if address == "" {
		return "", "", false
	}

	// DOMAIN-KEYWORD 是子串匹配, 不能去掉前导点, 也不能转成后缀匹配。
	if kind == "domain_keyword" {
		return kind, strings.ToLower(address), true
	}

	address = strings.ToLower(strings.TrimSuffix(address, "."))

	// 上游 AdGuard 列表里存在 DOMAIN,1.2.3.4 这种把 IP 当域名写的情况,
	// sing-box 的 domain 匹配不到 IP, 归到 ip_cidr 才有意义。
	if canonical, ok := canonicalIPOrCIDR(address); ok {
		return "ip_cidr", canonical, true
	}

	switch {
	case strings.HasPrefix(address, "+."):
		suffix := strings.TrimPrefix(address, "+.")
		if suffix == "" {
			return "", "", false
		}
		return "domain_suffix", suffix, true
	case strings.HasPrefix(address, "."):
		if len(address) == 1 {
			return "", "", false
		}
		// sing-box 的 domain_suffix 用前导点表示只匹配子域, 与 Clash 的
		// ".example.com" 语义一致, 因此保留前导点。
		return "domain_suffix", address, true
	case strings.Contains(address, "*"):
		pattern := wildcardToRegexp(address)
		if !validRegex(pattern) {
			return "", "", false
		}
		return "domain_regex", pattern, true
	default:
		return kind, address, true
	}
}

// wildcardToRegexp 把 Clash 的 * 通配写法转成 sing-box 的 domain_regex。
// Clash 的 * 只匹配一级(不跨过点), 因此每个标签转换为 [^.]+。
func wildcardToRegexp(rule string) string {
	labels := strings.Split(rule, ".")
	for index, label := range labels {
		labels[index] = wildcardLabelToRegexp(label)
	}
	return "^" + strings.Join(labels, `\.`) + "$"
}

// wildcardLabelToRegexp 把域名里的 * 通配转成只匹配单个标签的正则片段。
func wildcardLabelToRegexp(label string) string {
	if label == "*" {
		return `[^.]+`
	}
	parts := strings.Split(label, "*")
	for index := range parts {
		parts[index] = regexp.QuoteMeta(parts[index])
	}
	return strings.Join(parts, `[^.]*`)
}

// pathWildcardToRegexp 把 mihomo 的 PROCESS-PATH-WILDCARD 转成正则。
// 进程路径里的 * 可以跨过 / 等任意字符, 因此使用 .* 而不是 [^.]*。
func pathWildcardToRegexp(pattern string) string {
	parts := strings.Split(pattern, "*")
	for index := range parts {
		parts[index] = regexp.QuoteMeta(parts[index])
	}
	return "^" + strings.Join(parts, `.*`) + "$"
}

func validRegex(pattern string) bool {
	_, err := regexp.Compile(pattern)
	return err == nil
}

// canonicalIPOrCIDR 统一 IP/CIDR 的写法, 既能接受 1.2.3.4, 也能接受 1.2.3.0/24。
// 裸 IP 会补成 /32 或 /128, 便于和 IP-CIDR 条目去重。
func canonicalIPOrCIDR(value string) (string, bool) {
	if prefix, err := netip.ParsePrefix(value); err == nil {
		return prefix.Masked().String(), true
	}
	if addr, err := netip.ParseAddr(value); err == nil {
		return netip.PrefixFrom(addr, addr.BitLen()).String(), true
	}
	return "", false
}

func decodePayload(body []byte) ([]string, error) {
	var document any
	if err := yaml.Unmarshal(body, &document); err != nil {
		return nil, err
	}

	var rawPayload []any
	switch value := document.(type) {
	case map[string]any:
		rawPayload, _ = value["payload"].([]any)
	case map[any]any:
		rawPayload, _ = value["payload"].([]any)
	case []any:
		// 少数规则文件直接是一个 YAML 列表, 没有 payload: 包装。
		rawPayload = value
	}
	if rawPayload == nil {
		return nil, nil
	}

	payload := make([]string, 0, len(rawPayload))
	for _, item := range rawPayload {
		ruleInfo, ok := item.(string)
		if !ok {
			continue
		}
		ruleInfo = strings.TrimSpace(strings.Trim(ruleInfo, `"'`))
		if ruleInfo != "" {
			payload = append(payload, ruleInfo)
		}
	}
	return payload, nil
}

func (f *file) parseOriginFile() (ruleMap map[string][]string, err error) {
	switch strings.ToLower(f.fileSuffix) {
	case "yaml", "yml":
	default:
		return nil, errors.New("the file format is not yaml")
	}

	p := newParser()
	bodies := append([][]byte{f.fileBody}, f.extraBodies...)
	for _, body := range bodies {
		payload, decodeErr := decodePayload(body)
		if decodeErr != nil {
			return nil, fmt.Errorf("decode %v: %w", f.fullFileName, decodeErr)
		}
		for _, ruleInfo := range payload {
			p.add(ruleInfo)
		}
	}

	for _, ruleType := range sortedKeys(p.unsupported) {
		log.Printf("unsupported rule type %v x%d in %v", ruleType, p.unsupported[ruleType], f.fullFileName)
	}
	for _, ruleType := range sortedKeys(p.invalid) {
		log.Printf("invalid %v rule x%d in %v", ruleType, p.invalid[ruleType], f.fullFileName)
	}
	if len(p.ruleMap) == 0 {
		return nil, fmt.Errorf("%w in %v", errNoValidRule, f.fullFileName)
	}
	return p.ruleMap, nil
}

func sortedKeys(values map[string]int) []string {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}
