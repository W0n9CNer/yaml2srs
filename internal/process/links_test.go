package process

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestParseLink(t *testing.T) {
	tests := []struct {
		name       string
		link       string
		wantName   string
		wantSuffix string
		wantErr    bool
	}{
		{
			name:       "plain yaml",
			link:       "https://example.com/rule/Baidu.yaml",
			wantName:   "Baidu",
			wantSuffix: "yaml",
		},
		{
			name:       "query string",
			link:       "https://example.com/rule/Baidu.yaml?token=abc",
			wantName:   "Baidu",
			wantSuffix: "yaml",
		},
		{
			name:       "uppercase extension",
			link:       "https://example.com/Baidu.YML",
			wantName:   "Baidu",
			wantSuffix: "yml",
		},
		{
			name:    "no extension",
			link:    "https://example.com/rules",
			wantErr: true,
		},
		{
			name:    "unsupported extension",
			link:    "https://example.com/rules.txt",
			wantErr: true,
		},
		{
			name:    "unsupported scheme",
			link:    "ftp://example.com/Baidu.yaml",
			wantErr: true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, err := parseLink(test.link)
			if test.wantErr {
				if err == nil {
					t.Fatalf("parseLink(%q) error = nil, want error", test.link)
				}
				return
			}
			if err != nil {
				t.Fatalf("parseLink(%q) error = %v", test.link, err)
			}
			if got.fileName != test.wantName || got.fileSuffix != test.wantSuffix {
				t.Fatalf("parseLink(%q) = %#v, want name %q suffix %q",
					test.link, got, test.wantName, test.wantSuffix)
			}
		})
	}
}

func TestLinksScan(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		_, _ = writer.Write([]byte("payload:\n  - DOMAIN,example.com\n"))
	}))
	defer server.Close()

	dir := t.TempDir()
	linksPath := filepath.Join(dir, "links.txt")
	content := strings.Join([]string{
		"# comment",
		"",
		server.URL + "/Baidu.yaml",
		server.URL + "/Baidu.yaml?token=duplicate",
		server.URL + "/rules",
	}, "\n")
	if err := os.WriteFile(linksPath, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	ch := make(chan *file, 16)
	scanErr := newLinks(&basePath{path: linksPath}).scan(ch)
	if scanErr == nil {
		t.Fatal("scan() error = nil, want an error for the extensionless link")
	}
	if !strings.Contains(scanErr.Error(), "unsupported file suffix") {
		t.Fatalf("scan() error = %v, want unsupported file suffix", scanErr)
	}

	var files []*file
	for got := range ch {
		files = append(files, got)
	}
	if len(files) != 1 {
		t.Fatalf("got %d files, want 1 (duplicate name should be skipped)", len(files))
	}
	if files[0].fileName != "Baidu" || files[0].fileSuffix != "yaml" {
		t.Fatalf("unexpected file: %#v", files[0])
	}
}
