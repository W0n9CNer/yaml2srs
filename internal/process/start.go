package process

import (
	"errors"
	"fmt"
	"log"
)

const (
	CategoryFolder = "folder"
	CategoryLinks  = "links"
)

// Start 生成规则集。只有部分文件处理失败时也会返回错误, 让调用方(CLI/CI)能感知到。
func Start(path string, outputPath string, category string) error {
	if path == "" {
		return errors.New("path is required")
	}
	if outputPath == "" {
		return errors.New("output path is required")
	}

	basePath := newBasePath(
		withPath(path),
		withOutputPath(outputPath),
	)

	var s scan
	switch category {
	case CategoryFolder:
		s = newFolder(basePath)
	case CategoryLinks:
		s = newLinks(basePath)
	default:
		return fmt.Errorf("unknown category %q, only %q or %q", category, CategoryFolder, CategoryLinks)
	}

	ch := make(chan *file, 16)
	st := new(stats)

	// 先启动消费者, 再开始扫描/下载
	wg := listen(ch, st)
	scanErr := s.scan(ch)
	wg.Wait()

	if scanErr != nil || st.failed.Load() > 0 {
		log.Printf("finished with errors: %d rule sets generated, %d skipped, %d failed",
			st.generated.Load(), st.skipped.Load(), st.failed.Load())
	} else {
		log.Printf("done: %d rule sets generated, %d skipped, %d failed",
			st.generated.Load(), st.skipped.Load(), st.failed.Load())
	}

	if scanErr != nil {
		return scanErr
	}
	if st.failed.Load() > 0 {
		return fmt.Errorf("%d rule set(s) failed, see log for details", st.failed.Load())
	}
	if st.generated.Load() == 0 {
		return errors.New("no rule set generated")
	}
	return nil
}
