package process

import (
	"errors"
	"fmt"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

type folder struct {
	p *basePath
}

func newFolder(p *basePath) scan {
	return &folder{
		p: p,
	}
}

// scan 遍历规则目录, 逐个读取 yaml 规则文件并投递到管道。
// 上游 ios_rule_script 中存在同名文件(例如 Direct.yaml 同时出现在 Direct 和 AdGuardSDNSFilter 下),
// 因此先按路径排序, 同名文件只保留排序在前的那个, 保证每次生成的结果一致。
// isResolveVariantName 判断文件名是否是 Clash fake-ip 专用的 Resolve 变体。
// 上游的变体名形如 Xiaomi_Resolve.yaml、Xiaomi_No_Resolve.yaml, 只有这类
// 以 _Resolve / _No_Resolve 结尾的文件才需要跳过。
func isResolveVariantName(lowerName string) bool {
	return strings.HasSuffix(lowerName, "_resolve.yaml") ||
		strings.HasSuffix(lowerName, "_resolve.yml") ||
		strings.HasSuffix(lowerName, "_no_resolve.yaml") ||
		strings.HasSuffix(lowerName, "_no_resolve.yml")
}

func (f *folder) scan(ch chan *file) error {
	defer close(ch)

	var paths []string
	var scanErrors []error
	walkErr := filepath.WalkDir(f.p.path, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			scanErrors = append(scanErrors, fmt.Errorf("walk %v: %w", path, err))
			return nil
		}
		if d.IsDir() {
			return nil
		}
		lowerName := strings.ToLower(d.Name())
		if !strings.HasSuffix(lowerName, ".yaml") && !strings.HasSuffix(lowerName, ".yml") {
			return nil
		}
		// 上游以 _Resolve.yaml / _No_Resolve.yaml 结尾的是 Clash fake-ip 专用变体,
		// 与普通版本内容重复, 不参与生成。只按后缀判断, 避免误伤名字里带
		// "resolve" 的正常规则集。
		if isResolveVariantName(lowerName) {
			return nil
		}
		paths = append(paths, path)
		return nil
	})
	if walkErr != nil {
		scanErrors = append(scanErrors, fmt.Errorf("walk root %v: %w", f.p.path, walkErr))
	}

	// 先按路径深度、再按字典序排序: 上游重名规则优先取层级更浅的那个
	// (例如 Direct/Direct.yaml 优先于 AdGuardSDNSFilter/Direct/Direct.yaml),
	// 保证每次生成的取舍结果一致。
	sort.Slice(paths, func(i, j int) bool {
		depthI := strings.Count(filepath.Clean(paths[i]), string(filepath.Separator))
		depthJ := strings.Count(filepath.Clean(paths[j]), string(filepath.Separator))
		if depthI != depthJ {
			return depthI < depthJ
		}
		return paths[i] < paths[j]
	})

	if len(paths) == 0 {
		scanErrors = append(scanErrors, fmt.Errorf("no yaml rule file found in %v", f.p.path))
	}

	seen := make(map[string]string, len(paths))
	for _, path := range paths {
		fullFileName := filepath.Base(path)
		extension := filepath.Ext(fullFileName)
		fileName := strings.TrimSuffix(fullFileName, extension)
		if fileName == "" || fileName == "." || fileName == ".." {
			log.Printf("Unrecognized file name path %v", path)
			continue
		}

		if prev, ok := seen[fileName]; ok {
			log.Printf("duplicate rule name %v, keep %v skip %v", fileName, prev, path)
			continue
		}
		seen[fileName] = path

		body, err := os.ReadFile(path)
		if err != nil {
			scanErrors = append(scanErrors, fmt.Errorf("read %v: %w", path, err))
			continue
		}
		options := []fileOption{
			withFullFileName(fullFileName),
			withFileName(fileName),
			withFileSuffix(strings.TrimPrefix(extension, ".")),
			withFileBody(body),
			withFileOutputPath(f.p),
		}

		// 上游部分规则的 <名称>.yaml 只留了关键字/进程名等条目, 完整的域名与 IP 列表在
		// <名称>_Classical.yaml 里(例如 ChinaMax 27 vs 123062 条), 这里把两份合并,
		// 否则生成的 ChinaMax 之类规则集几乎是空的。
		// 主文件扩展名大小写不固定, 统一按小写后缀尝试 .yaml / .yml。
		lowerExtension := strings.ToLower(extension)
		classicalNames := []string{fileName + "_Classical" + lowerExtension}
		if lowerExtension != ".yaml" {
			classicalNames = append(classicalNames, fileName+"_Classical.yaml")
		}
		if lowerExtension != ".yml" {
			classicalNames = append(classicalNames, fileName+"_Classical.yml")
		}
		for _, classicalName := range classicalNames {
			classicalPath := filepath.Join(filepath.Dir(path), classicalName)
			extraBody, readErr := os.ReadFile(classicalPath)
			if readErr == nil {
				options = append(options, withExtraBody(extraBody))
				log.Printf("merge %v into %v", classicalName, fullFileName)
				break
			}
			if !errors.Is(readErr, fs.ErrNotExist) {
				log.Printf("read classical file %v err %v", classicalPath, readErr)
			}
		}

		ch <- newFile(options...)
	}

	return errors.Join(scanErrors...)
}
