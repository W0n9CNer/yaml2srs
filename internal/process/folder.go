package process

import (
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

// scan 遍历规则目录, 逐个读取 yaml 规则文件并投递到管道.
// 上游 ios_rule_script 中存在同名文件(例如 Direct.yaml 同时出现在 Direct 和 AdGuardSDNSFilter 下),
// 因此先按路径排序, 同名文件只保留排序在前的那个, 保证每次生成的结果一致.
func (f *folder) scan(ch chan *file) {
	defer close(ch)

	var paths []string
	walkErr := filepath.WalkDir(f.p.path, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			log.Printf("WalkDir err %v path %v", err.Error(), path)
			return nil
		}
		if d.IsDir() {
			return nil
		}
		lowerName := strings.ToLower(d.Name())
		if strings.HasSuffix(lowerName, ".yaml") && !strings.Contains(lowerName, "resolve") {
			paths = append(paths, path)
		}
		return nil
	})
	if walkErr != nil {
		log.Printf("WalkDir root err %v path %v", walkErr.Error(), f.p.path)
	}

	// 先按路径深度、再按字典序排序: 上游重名规则优先取层级更浅的那个
	// (例如 Direct/Direct.yaml 优先于 AdGuardSDNSFilter/Direct/Direct.yaml),
	// 保证每次生成的取舍结果一致
	sort.Slice(paths, func(i, j int) bool {
		depthI, depthJ := strings.Count(paths[i], "/"), strings.Count(paths[j], "/")
		if depthI != depthJ {
			return depthI < depthJ
		}
		return paths[i] < paths[j]
	})

	seen := make(map[string]string, len(paths))
	for _, path := range paths {
		fullFileName := filepath.Base(path)

		nameSplit := strings.Split(fullFileName, ".")
		if len(nameSplit) != 2 {
			log.Printf("Unrecognized file name path %v", path)
			continue
		}

		fileName := nameSplit[0]
		if prev, ok := seen[fileName]; ok {
			log.Printf("duplicate rule name %v, keep %v skip %v", fileName, prev, path)
			continue
		}
		seen[fileName] = path

		body, err := os.ReadFile(path)
		if err != nil {
			log.Printf("ReadFile err %v path %v", err.Error(), path)
			continue
		}
		options := []fileOption{
			withFullFileName(fullFileName),
			withFileName(fileName),
			withFileSuffix(nameSplit[1]),
			withFileBody(body),
			withFileOutputPath(f.p),
		}

		// 上游部分规则的 <名称>.yaml 只留了关键字/进程名等条目, 完整的域名与 IP 列表在
		// <名称>_Classical.yaml 里(例如 ChinaMax 27 vs 123062 条), 这里把两份合并,
		// 否则生成的 ChinaMax 之类规则集几乎是空的
		classicalPath := filepath.Join(filepath.Dir(path), fileName+"_Classical.yaml")
		if extraBody, readErr := os.ReadFile(classicalPath); readErr == nil {
			options = append(options, withExtraBody(extraBody))
			log.Printf("merge %v into %v", filepath.Base(classicalPath), fullFileName)
		}

		ch <- newFile(options...)
	}
}
