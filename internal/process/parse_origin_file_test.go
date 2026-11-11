package process

import (
	"errors"
	"reflect"
	"testing"
)

func parseTestFile(t *testing.T, suffix string, bodies ...string) (map[string][]string, error) {
	t.Helper()
	options := []fileOption{
		withFullFileName("Test." + suffix),
		withFileName("Test"),
		withFileSuffix(suffix),
		withFileBody([]byte(bodies[0])),
	}
	for _, body := range bodies[1:] {
		options = append(options, withExtraBody([]byte(body)))
	}
	return newFile(options...).parseOriginFile()
}

func TestParseOriginFile(t *testing.T) {
	tests := []struct {
		name     string
		payload  string
		expected map[string][]string
	}{
		{
			name: "typed rules",
			payload: `payload:
  - DOMAIN,exact.com
  - DOMAIN-SUFFIX,example.com
  - DOMAIN-KEYWORD,keyword
  - DOMAIN-REGEX,^a{2,3}\.com$
  - DOMAIN-WILDCARD,*.google.com
  - IP-CIDR,10.0.0.0/8
  - IP-CIDR6,2001:db8::/32
`,
			expected: map[string][]string{
				"domain":         {"exact.com"},
				"domain_suffix":  {"example.com"},
				"domain_keyword": {"keyword"},
				"domain_regex":   {`^a{2,3}\.com$`},
				"ip_cidr":        {"10.0.0.0/8", "2001:db8::/32"},
			},
		},
		{
			name: "bare wildcard forms",
			payload: `payload:
  - ".lan"
  - "+.xboxlive.com"
  - "*.star.com"
  - "www.*.mid.com"
  - "xbox.*.microsoft.com"
  - "foo*bar.com"
  - "plain.example.com"
`,
			expected: map[string][]string{
				"domain_suffix": {".lan", "xboxlive.com"},
				"domain_regex": {
					`^[^.]+\.star\.com$`,
					`^www\.[^.]+\.mid\.com$`,
					`^xbox\.[^.]+\.microsoft\.com$`,
					`^foo[^.]*bar\.com$`,
				},
				"domain": {"plain.example.com"},
			},
		},
		{
			name: "uppercase and trailing dot",
			payload: `payload:
  - DOMAIN-SUFFIX,Example.COM.
  - DOMAIN,Upper.Example.COM
  - DOMAIN-KEYWORD,.Tmall.COM
`,
			expected: map[string][]string{
				"domain_suffix":  {"example.com"},
				"domain":         {"upper.example.com"},
				"domain_keyword": {".tmall.com"},
			},
		},
		{
			name: "ip written as domain",
			payload: `payload:
  - DOMAIN,1.2.3.4
  - DOMAIN,2001:DB8::1
  - DOMAIN-SUFFIX,10.0.0.0/8
  - 1.2.3.4
  - 10.0.0.0/8
`,
			expected: map[string][]string{
				"ip_cidr": {
					"1.2.3.4/32",
					"2001:db8::1/128",
					"10.0.0.0/8",
				},
			},
		},
		{
			name: "process rules",
			payload: `payload:
  - PROCESS-NAME,ToDesk.exe
  - PROCESS-PATH,C:\Program Files\ToDesk\ToDesk.exe
  - PROCESS-PATH,C:\Program Files\a,b.exe
  - PROCESS-PATH-REGEX,^C:\\\\Tools\\\\.*$
`,
			expected: map[string][]string{
				"process_name":       {"ToDesk.exe"},
				"process_path":       {`C:\Program Files\ToDesk\ToDesk.exe`, `C:\Program Files\a,b.exe`},
				"process_path_regex": {`^C:\\\\Tools\\\\.*$`},
			},
		},
		{
			name: "rule options are ignored",
			payload: `payload:
  - DOMAIN-SUFFIX,example.com,no-resolve
  - IP-CIDR,192.168.1.1,no-resolve
`,
			expected: map[string][]string{
				"domain_suffix": {"example.com"},
				"ip_cidr":       {"192.168.1.1/32"},
			},
		},
		{
			name: "ip suffix and ports",
			payload: `payload:
  - IP-SUFFIX,8.8.8.8/24
  - SRC-IP-CIDR,192.168.1.201/32
  - SRC-IP-SUFFIX,10.1.2.3/8
  - DST-PORT,80
  - DST-PORT,114-514/810-1919,65530
  - SRC-PORT,7777
  - SRC-PORT,1000-2000
  - PROCESS-PATH-WILDCARD,/usr/*/wget
`,
			expected: map[string][]string{
				"ip_cidr":        {"8.8.8.0/24"},
				"source_ip_cidr": {"192.168.1.201/32", "10.0.0.0/8"},
				"port":           {"80", "65530"},
				"port_range":     {"114:514", "810:1919"},
				"source_port":    {"7777"},
				"source_port_range": {
					"1000:2000",
				},
				"process_path_regex": {`^/usr/.*/wget$`},
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, err := parseTestFile(t, "yaml", test.payload)
			if err != nil {
				t.Fatalf("parseOriginFile() error = %v", err)
			}
			if !reflect.DeepEqual(got, test.expected) {
				t.Fatalf("parseOriginFile() = %#v, want %#v", got, test.expected)
			}
		})
	}
}

func TestParseOriginFileDeduplicates(t *testing.T) {
	payload := `payload:
  - DOMAIN,example.com
  - DOMAIN,EXAMPLE.COM
  - DOMAIN-SUFFIX,example.com
  - DOMAIN-SUFFIX,EXAMPLE.COM
`
	got, err := parseTestFile(t, "yaml", payload)
	if err != nil {
		t.Fatal(err)
	}
	expected := map[string][]string{
		"domain":        {"example.com"},
		"domain_suffix": {"example.com"},
	}
	if !reflect.DeepEqual(got, expected) {
		t.Fatalf("got %#v, want %#v", got, expected)
	}
}

func TestParseOriginFileMergesClassical(t *testing.T) {
	main := `payload:
  - DOMAIN-KEYWORD,keyword
`
	classical := `payload:
  - DOMAIN-SUFFIX,example.com
  - DOMAIN-SUFFIX,example.com
  - DOMAIN,exact.com
`
	got, err := parseTestFile(t, "yaml", main, classical)
	if err != nil {
		t.Fatal(err)
	}
	expected := map[string][]string{
		"domain_keyword": {"keyword"},
		"domain_suffix":  {"example.com"},
		"domain":         {"exact.com"},
	}
	if !reflect.DeepEqual(got, expected) {
		t.Fatalf("got %#v, want %#v", got, expected)
	}
}

func TestParseOriginFileAcceptsYml(t *testing.T) {
	got, err := parseTestFile(t, "yml", "payload:\n  - DOMAIN,example.com\n")
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(got, map[string][]string{"domain": {"example.com"}}) {
		t.Fatalf("unexpected result: %#v", got)
	}
}

func TestParseOriginFileNoValidRule(t *testing.T) {
	tests := []struct {
		name    string
		payload string
	}{
		{name: "unsupported only", payload: "payload:\n  - GEOIP,CN\n  - IP-ASN,13335\n"},
		{name: "domain wildcard only", payload: "payload:\n  - DOMAIN-WILDCARD,*.google.com\n"},
		{name: "invalid port", payload: "payload:\n  - DST-PORT,70000\n"},
		{name: "invalid regex", payload: "payload:\n  - DOMAIN-REGEX,[\n"},
		{name: "empty payload", payload: "payload: []\n"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, err := parseTestFile(t, "yaml", test.payload)
			if !errors.Is(err, errNoValidRule) {
				t.Fatalf("error = %v, want errNoValidRule", err)
			}
		})
	}
}

func TestParseOriginFileRejectsNonYaml(t *testing.T) {
	_, err := parseTestFile(t, "json", `{"payload":[]}`)
	if err == nil || errors.Is(err, errNoValidRule) {
		t.Fatalf("error = %v, want file format error", err)
	}
}
