package process

import (
	"errors"
	"log"
	"sync"
	"sync/atomic"

	"github.com/W0n9CNer/yaml2srs/internal/outputor"
)

type stats struct {
	total     atomic.Int64
	generated atomic.Int64
	skipped   atomic.Int64
	failed    atomic.Int64
}

func listen(f chan *file, st *stats) *sync.WaitGroup {
	wg := sync.WaitGroup{}
	// 开启多个协程监听通道
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go worker(f, &wg, st)
	}
	return &wg
}

func worker(f chan *file, wg *sync.WaitGroup, st *stats) {
	defer wg.Done()
	for v := range f {
		st.total.Add(1)

		data, err := v.parseOriginFile()
		if err != nil {
			if errors.Is(err, errNoValidRule) {
				log.Printf("skip %v: %v", v.fullFileName, err)
				st.skipped.Add(1)
				continue
			}
			log.Printf("parseOriginFile err %v", err)
			st.failed.Add(1)
			continue
		}

		output := outputor.NewOutput(
			outputor.WithFileName(v.fileName),
			outputor.WithOutputPath(v.filePath.outputPath),
			outputor.WithRuleMap(data),
			outputor.WithRawFile(v.fullFileName, v.fileBody),
		)
		if err = output.Output(); err != nil {
			log.Printf("output %v err %v", v.fullFileName, err)
			st.failed.Add(1)
			continue
		}
		st.generated.Add(1)
	}
}
