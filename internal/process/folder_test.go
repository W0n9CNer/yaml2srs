package process

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestFolderScan(t *testing.T) {
	dir := t.TempDir()
	write := func(name, content string) {
		t.Helper()
		path := filepath.Join(dir, name)
		if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(content), 0644); err != nil {
			t.Fatal(err)
		}
	}

	write("Rule.yaml", "payload:\n  - DOMAIN-KEYWORD,keyword\n")
	write("Rule_Classical.yaml", "payload:\n  - DOMAIN,exact.com\n")
	write("Rule_No_Resolve.yaml", "payload:\n  - DOMAIN,ignored.com\n")
	write("Rule_Resolve.yaml", "payload:\n  - DOMAIN,ignored2.com\n")
	write("Resolver.yaml", "payload:\n  - DOMAIN,resolver.com\n")
	write("Other.yml", "payload:\n  - DOMAIN,other.com\n")
	write("Other_Classical.yaml", "payload:\n  - DOMAIN,other-classical.com\n")
	write("Nested/Direct.yaml", "payload:\n  - DOMAIN,nested.com\n")
	write("Direct/Direct.yaml", "payload:\n  - DOMAIN,root.com\n")

	ch := make(chan *file, 16)
	if err := newFolder(&basePath{path: dir}).scan(ch); err != nil {
		t.Fatalf("scan() error = %v", err)
	}

	files := make(map[string]*file)
	for got := range ch {
		files[got.fileName] = got
	}
	if len(files) != 6 {
		t.Fatalf("got %d files, want 6: %v", len(files), files)
	}
	if _, ok := files["Rule_No_Resolve"]; ok {
		t.Fatal("No_Resolve variant should be skipped")
	}
	if _, ok := files["Rule_Resolve"]; ok {
		t.Fatal("Resolve variant should be skipped")
	}
	// 名字里带 resolve 但不是 Resolve 变体的规则集不能被误跳过。
	if _, ok := files["Resolver"]; !ok {
		t.Fatal("Resolver should be kept")
	}
	if len(files["Rule"].extraBodies) != 1 {
		t.Fatalf("Rule extraBodies = %d, want 1", len(files["Rule"].extraBodies))
	}
	// 主文件是 .yml 时, 也要能合并 .yaml 形式的 Classical 文件。
	if len(files["Other"].extraBodies) != 1 {
		t.Fatalf("Other extraBodies = %d, want 1", len(files["Other"].extraBodies))
	}
	otherRules, err := files["Other"].parseOriginFile()
	if err != nil {
		t.Fatal(err)
	}
	expectedOther := map[string][]string{"domain": {"other.com", "other-classical.com"}}
	if !reflect.DeepEqual(otherRules, expectedOther) {
		t.Fatalf("Other = %#v, want %#v", otherRules, expectedOther)
	}
	if _, ok := files["Rule_Classical"]; !ok {
		t.Fatal("Classical file should still be generated as its own rule set")
	}
	if files["Other"].fileSuffix != "yml" {
		t.Fatalf("Other suffix = %q, want yml", files["Other"].fileSuffix)
	}

	// Direct 同名时保留层级更浅的那个。
	ruleMap, err := files["Direct"].parseOriginFile()
	if err != nil {
		t.Fatal(err)
	}
	expected := map[string][]string{"domain": {"root.com"}}
	if !reflect.DeepEqual(ruleMap, expected) {
		t.Fatalf("Direct = %#v, want %#v", ruleMap, expected)
	}
}
