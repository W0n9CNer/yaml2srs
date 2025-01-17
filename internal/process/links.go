package process

import (
	"bufio"
	"log"
	"os"
	pathBase "path"
	"strings"
	"sync"
	"time"
	"yaml2srs/tools"

	"github.com/go-resty/resty/v2"
)

type links struct {
	p *basePath
}

func newLinks(p *basePath) scan {
	return &links{
		p: p,
	}
}

func (f *links) scan(ch chan *file) {
	defer close(ch)

	if !strings.HasSuffix(f.p.path, ".txt") {
		log.Println("links path invalid")
		return
	}

	file, err := os.Open(f.p.path)
	if err != nil {
		log.Printf("open file %v err %v\n", f.p.path, err)
		return
	}
	defer file.Close()

	var links []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		links = append(links, line)
	}
	if len(links) == 0 {
		log.Println("no line found in links file")
		return
	}

	// 去重
	links = tools.UnorderedDeduplication(links)

	wg := sync.WaitGroup{}

	for _, link := range links {
		wg.Add(1)
		go func() {
			defer wg.Done()
			// 获取文件名后缀
			fullFileName := pathBase.Base(link)

			fullFileNameSplit := strings.Split(fullFileName, ".")

			if len(fullFileNameSplit) != 2 {
				log.Printf("file %v suffix invalid\n", link)
			}

			log.Printf("start download file %v\n", link)

			client := resty.New()
			resp, err := client.SetTimeout(60 * time.Second).R().Get(link)
			if err != nil {
				log.Printf("download file from %v err %v\n", link, err.Error())
				return
			}
			if resp.StatusCode() != 200 {
				log.Printf("download file from %v statsuCode %v\n", link, resp.StatusCode())
				return
			}
			ch <- newFile(
				withFullFileName(fullFileName),
				withFileName(fullFileNameSplit[0]),
				withFileSuffix(fullFileNameSplit[1]),
				withFileBody(resp.Body()),
				withFileOutputPath(f.p),
			)
		}()
	}
	wg.Wait()
}
