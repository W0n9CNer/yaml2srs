package process

import (
	"bytes"
	"errors"
	"regexp"
	"strings"

	"gopkg.in/yaml.v2"
)

type rule struct {
	Pattern string   `json:"pattern"`
	Address []string `json:"address"`
	Other   string   `json:"other"`
}

var mapDictMap = map[string]string{
	"DOMAIN":         "domain",
	"HOST":           "domain",
	"host":           "domain",
	"DOMAIN-SUFFIX":  "domain_suffix",
	"HOST-SUFFIX":    "domain_suffix",
	"DOMAIN-KEYWORD": "domain_keyword",
	"HOST-KEYWORD":   "domain_keyword",
	"host-keyword":   "domain_keyword",
	"URL-REGEX":      "domain_regex",
	"SRC-IP-CIDR":    "source_ip_cidr",
	"IP-CIDR":        "ip_cidr",
	"ip-cidr":        "ip_cidr",
	"IP-CIDR6":       "ip_cidr",
	"IP6-CIDR":       "ip_cidr",
	"SRC-PORT":       "source_port",
	"DST-PORT":       "port",
}

func (f *file) isIPv4OrIPv6(address string) string {
	// 使用正则表达式简化 IPv4 和 IPv6 的检测
	ipv4Pattern := `^(?:[0-9]{1,3}\.){3}[0-9]{1,3}$`
	ipv6Pattern := `([0-9a-fA-F]{1,4}:){7}([0-9a-fA-F]{1,4})`

	ipv4Match, _ := regexp.MatchString(ipv4Pattern, address)
	ipv6Match, _ := regexp.MatchString(ipv6Pattern, address)

	if ipv4Match {
		return "ipv4"
	} else if ipv6Match {
		return "ipv6"
	}
	return ""
}
func (f *file) parseOriginFile() (ruleMap map[string][]string, err error) {
	var rows []rule
	var yamlData map[string]interface{}

	if f.fileSuffix != "yaml" {
		err = errors.New("the file format is not yaml")
		return
	}
	decoder := yaml.NewDecoder(bytes.NewReader(f.fileBody))
	err = decoder.Decode(&yamlData)
	if err != nil {
		return
	}
	payload, ok := yamlData["payload"].([]interface{})
	if ok {
		for _, item := range payload {
			address := item.(string)
			address = strings.Trim(address, "'")
			var pattern string

			if !strings.Contains(address, ",") {
				if f.isIPv4OrIPv6(address) != "" {
					pattern = "IP-CIDR"
				} else {
					if strings.HasPrefix(address, "+") {
						pattern = "DOMAIN-SUFFIX"
						address = address[1:]
						if strings.HasPrefix(address, ".") {
							address = address[2:]
						}
					} else {
						pattern = "DOMAIN"
					}
				}
			} else {
				parts := strings.Split(address, ",")
				pattern = strings.TrimSpace(parts[0])
				address = strings.TrimSpace(parts[1])
			}
			rows = append(rows, rule{Pattern: pattern, Address: []string{address}})
		}
	}

	var filteredRows []rule
	for _, row := range rows {
		if _, ok := mapDictMap[row.Pattern]; ok {
			row.Pattern = mapDictMap[row.Pattern]
			filteredRows = append(filteredRows, row)
		}
	}
	// 组装成map
	ruleMap = make(map[string][]string)
	for _, v := range filteredRows {
		if _, ok := ruleMap[v.Pattern]; ok {
			ruleMap[v.Pattern] = append(ruleMap[v.Pattern], v.Address...)
		} else {
			ruleMap[v.Pattern] = v.Address
		}
	}
	return
}
