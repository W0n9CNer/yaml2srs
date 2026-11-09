package process

import (
	"bytes"
	"errors"
	"log"
	"net"
	"strings"

	"gopkg.in/yaml.v2"
)

// KEY: https://wiki.metacubex.one/config/rules/
// VALUE: https://sing-box.sagernet.org/configuration/route/rule/
// Clash 规则类型与 sing-box 规则集的对应关系
var mapDictMap = map[string]string{
	"DOMAIN":         "domain",
	"DOMAIN-SUFFIX":  "domain_suffix",
	"DOMAIN-KEYWORD": "domain_keyword",
	"DOMAIN-REGEX":   "domain_regex",
	// "GEOSITE":        "geosite",
	"IP-CIDR":  "ip_cidr",
	"IP-CIDR6": "ip_cidr",
	// "IP-SUFFIX":      "",
	// "IP-ASN":         "", // sing-box 1.11 起规则集不再支持 ip_asn
	// "GEOIP":     "geoip",
	// "SRC-GEOIP": "source_geoip",
	// "SRC-IP-ASN":     "",
	// "SRC-IP-CIDR": "source_ip_cidr",
	// "SRC-IP-SUFFIX":  "",
	// "DST-PORT": "port",
	// "SRC-PORT": "source_port",
	// "IN-PORT":        "",
	// "IN-TYPE":        "",
	// "IN-USER":        "",
	// "IN-NAME":        "",
	// "PROCESS-PATH":       "process_path",
	// "PROCESS-PATH-REGEX": "process_path_regex",
	// "PROCESS-NAME":       "process_name",
	// "PROCESS-NAME-REGEX": "",
	// "UID":                "user_id",
	// "NETWORK":            "network",
	// "DSCP":               "",
}

type parser struct {
	ruleMap     map[string][]string
	seen        map[string]map[string]struct{}
	unsupported map[string]int
}

func newParser() *parser {
	return &parser{
		ruleMap:     make(map[string][]string),
		seen:        make(map[string]map[string]struct{}),
		unsupported: make(map[string]int),
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

// add 把一条 Clash 规则(payload 中的一行)转换成 sing-box 规则项
func (p *parser) add(ruleInfo string) {
	var pattern string
	var address string

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

	if strings.Contains(ruleInfo, ",") {
		rules := strings.Split(ruleInfo, ",")
		k := strings.TrimSpace(rules[0])
		if v, ok := mapDictMap[k]; ok {
			pattern = v
			if k == "DOMAIN-REGEX" {
				// 正则本身可能包含逗号, 例如 ^a{2,3}\.com$, 需要整段拼接
				address = strings.Join(rules[1:], ",")
			} else if len(rules) > 1 {
				address = rules[1]
			}
		} else {
			p.unsupported[k]++
		}

	} else if strings.HasPrefix(ruleInfo, ".") {

		// 	通配符 . 可以一次性匹配多个级别
		// .baidu.com 匹配 tieba.baidu.com 和 123.tieba.baidu.com, 但不能匹配 baidu.com
		// 通配符 . 只能用于域名前缀匹配
		pattern = "domain_suffix"
		address = ruleInfo

	} else if strings.HasPrefix(ruleInfo, "+.") {

		// 通配符 ＋ 类似 DOMAIN-SUFFIX, 可以一次性匹配多个级别
		// ＋.baidu.com 匹配 tieba.baidu.com 和 123.tieba.baidu.com 或者 baidu.com
		// 通配符 ＋ 只能用于域名前缀匹配
		pattern = "domain_suffix"
		suffix, found := strings.CutPrefix(ruleInfo, "+.")

		if found {
			address = suffix
		}

	} else if strings.Contains(ruleInfo, "*") {

		// Clash 的通配符 * 一次只能匹配一级域名
		// *.baidu.com 只匹配 tieba.baidu.com 而不匹配 123.tieba.baidu.com 或者 baidu.com
		// *只匹配 localhost 等没有.的主机名

		pattern = "domain_regex"

		names := strings.Split(ruleInfo, ".")

		for index, value := range names {

			has := value == "*"

			// *.baidu.com

			if index == 0 && has {
				names[0] = "^[a-zA-Z0-9-]"
				continue

			} else {

				// www.*.baidu.com
				if has {
					names[index] = "[a-zA-Z0-9-]"
				}
			}
		}
		address = strings.Join(names, "+\\.") + "$"
	} else if _, _, err := net.ParseCIDR(ruleInfo); err == nil {
		pattern = "ip_cidr"
		address = ruleInfo
	} else {
		pattern = "domain"
		address = ruleInfo
	}

	p.appendRule(pattern, address)
}

func decodePayload(body []byte) ([]string, error) {
	var yamlData map[string]interface{}
	if err := yaml.NewDecoder(bytes.NewReader(body)).Decode(&yamlData); err != nil {
		return nil, err
	}
	rawPayload, ok := yamlData["payload"].([]interface{})
	if !ok {
		return nil, nil
	}
	payload := make([]string, 0, len(rawPayload))
	for _, item := range rawPayload {
		ruleInfo, ok := item.(string)
		if !ok {
			continue
		}
		ruleInfo = strings.TrimSpace(strings.Trim(ruleInfo, "'"))
		if ruleInfo != "" {
			payload = append(payload, ruleInfo)
		}
	}
	return payload, nil
}

func (f *file) parseOriginFile() (ruleMap map[string][]string, err error) {
	if strings.ToLower(f.fileSuffix) != "yaml" {
		err = errors.New("the file format is not yaml")
		return
	}

	p := newParser()
	bodies := append([][]byte{f.fileBody}, f.extraBodies...)
	for _, body := range bodies {
		payload, decodeErr := decodePayload(body)
		if decodeErr != nil {
			err = decodeErr
			return
		}
		for _, ruleInfo := range payload {
			p.add(ruleInfo)
		}
	}

	for k, count := range p.unsupported {
		log.Printf("unsupported rule type %v x%d in %v", k, count, f.fullFileName)
	}
	if len(p.ruleMap) == 0 {
		log.Printf("no valid rule found in %v", f.fullFileName)
	}
	return p.ruleMap, nil
}
