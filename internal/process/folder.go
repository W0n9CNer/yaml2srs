package process

import (
	"io/fs"
	"log"
	"os"
	"path/filepath"
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

func (f *folder) scan(ch chan *file) {
	defer close(ch)
	filepath.WalkDir(f.p.path, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			log.Printf("WalkDir err %v path %v", err.Error(), path)
			return nil
		}

		if !d.IsDir() {

			fullFileName := d.Name()

			if strings.HasSuffix(d.Name(), ".yaml") && !strings.Contains(strings.ToLower(fullFileName), "resolve") {
				nameSplit := strings.Split(d.Name(), ".")
				if len(nameSplit) != 2 {
					log.Printf("Unrecognized file name path %v", path+fullFileName)
					return nil
				}

				fileName := nameSplit[0]
				suffix := nameSplit[1]

				body, err := os.ReadFile(path)
				if err != nil {
					log.Printf("ReadFile err %v path %v", err.Error(), path+fullFileName)
				}
				ch <- newFile(
					withFullFileName(fullFileName),
					withFileName(fileName),
					withFileSuffix(suffix),
					withFileBody(body),
					withFileOutputPath(f.p),
				)
			}
		}
		return nil
	})
}
