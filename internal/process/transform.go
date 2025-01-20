package process

import (
	"log"
	"sync"

	"github.com/W0n9CNer/yaml2srs/internal/outputor"
)

func listen(f chan *file) *sync.WaitGroup {
	wg := sync.WaitGroup{}
	// 开启多个协程监听通道
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go worker(f, &wg)
	}
	return &wg
}

func worker(f chan *file, wg *sync.WaitGroup) {
	defer wg.Done()
	for v := range f {
		if data, err := v.parseOriginFile(); err != nil {
			log.Printf("parseOriginFile err %v", err.Error())
			continue
		} else {
			output := outputor.NewOutput(
				outputor.WithFileName(v.fileName),
				outputor.WithOutputPath(v.filePath.outputPath),
				outputor.WithRuleMap(data),
			)
			err := output.Output()
			if err != nil {
				log.Printf("output err %v", err.Error())
				continue
			}
		}
	}
}
