package process

import (
	"bufio"
	"errors"
	"fmt"
	"log"
	"net/url"
	"os"
	pathpkg "path"
	"strings"
	"sync"
	"time"

	"github.com/go-resty/resty/v2"
)

type links struct {
	p *basePath
}

type linkEntry struct {
	url        string
	fileName   string
	fileSuffix string
}

func newLinks(p *basePath) scan {
	return &links{
		p: p,
	}
}

func (f *links) scan(ch chan *file) error {
	defer close(ch)

	if !strings.HasSuffix(strings.ToLower(f.p.path), ".txt") {
		return fmt.Errorf("links path invalid: %v", f.p.path)
	}

	file, err := os.Open(f.p.path)
	if err != nil {
		return fmt.Errorf("open file %v: %w", f.p.path, err)
	}
	defer file.Close()

	// 先解析并去重, 再并发下载。这里按输出文件名去重, 而不是按 URL:
	// 两个不同 URL 可能都以 Baidu.yaml 结尾, 同时下载会写同一个输出文件。
	entries := make([]linkEntry, 0, 64)
	seen := make(map[string]string, 64)
	var scanErrors []error
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		entry, parseErr := parseLink(line)
		if parseErr != nil {
			scanErrors = append(scanErrors, parseErr)
			continue
		}
		if previous, ok := seen[entry.fileName]; ok {
			log.Printf("duplicate link name %v, keep %v skip %v", entry.fileName, previous, entry.url)
			continue
		}
		seen[entry.fileName] = entry.url
		entries = append(entries, entry)
	}
	if err = scanner.Err(); err != nil {
		scanErrors = append(scanErrors, fmt.Errorf("read links file %v: %w", f.p.path, err))
	}
	if len(entries) == 0 {
		scanErrors = append(scanErrors, fmt.Errorf("no valid link found in %v", f.p.path))
		return errors.Join(scanErrors...)
	}

	// resty client 是并发安全的, 整个批次复用同一个连接池。
	client := resty.New().
		SetTimeout(60 * time.Second).
		SetRetryCount(3).
		SetRetryWaitTime(2 * time.Second)

	const concurrency = 8
	sem := make(chan struct{}, concurrency)
	var wg sync.WaitGroup
	var lock sync.Mutex
	for _, entry := range entries {
		entry := entry
		wg.Add(1)
		sem <- struct{}{}
		go func() {
			defer wg.Done()
			defer func() { <-sem }()

			log.Printf("start download file %v", entry.url)
			resp, downloadErr := client.R().Get(entry.url)
			if downloadErr != nil {
				lock.Lock()
				scanErrors = append(scanErrors, fmt.Errorf("download %v: %w", entry.url, downloadErr))
				lock.Unlock()
				return
			}
			if resp.StatusCode() != 200 {
				lock.Lock()
				scanErrors = append(scanErrors, fmt.Errorf("download %v: status code %v", entry.url, resp.StatusCode()))
				lock.Unlock()
				return
			}
			if len(resp.Body()) == 0 {
				lock.Lock()
				scanErrors = append(scanErrors, fmt.Errorf("download %v: empty body", entry.url))
				lock.Unlock()
				return
			}
			ch <- newFile(
				withFullFileName(entry.fileName+"."+entry.fileSuffix),
				withFileName(entry.fileName),
				withFileSuffix(entry.fileSuffix),
				withFileBody(resp.Body()),
				withFileOutputPath(f.p),
			)
		}()
	}
	wg.Wait()

	return errors.Join(scanErrors...)
}

// parseLink 从规则链接里取出输出文件名。
// 不能直接用 path.Base(link): URL 可能带 ?token=... 查询串, 也可能没有扩展名。
func parseLink(raw string) (linkEntry, error) {
	parsed, err := url.Parse(raw)
	if err != nil {
		return linkEntry{}, fmt.Errorf("parse link %v: %w", raw, err)
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return linkEntry{}, fmt.Errorf("parse link %v: unsupported scheme %q", raw, parsed.Scheme)
	}

	fullFileName := pathpkg.Base(parsed.Path)
	extension := strings.ToLower(pathpkg.Ext(fullFileName))
	if extension != ".yaml" && extension != ".yml" {
		return linkEntry{}, fmt.Errorf("parse link %v: unsupported file suffix %q", raw, extension)
	}
	fileName := strings.TrimSuffix(fullFileName, pathpkg.Ext(fullFileName))
	if fileName == "" || fileName == "." || fileName == ".." ||
		strings.ContainsAny(fileName, `/\`) {
		return linkEntry{}, fmt.Errorf("parse link %v: invalid file name %q", raw, fileName)
	}

	return linkEntry{
		url:        raw,
		fileName:   fileName,
		fileSuffix: strings.TrimPrefix(extension, "."),
	}, nil
}
