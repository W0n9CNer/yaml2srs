package process

import "log"

func Start(path string, outputPath string, category string) {
	basePath := newBasePath(
		withPath(path),
		withOutputPath(outputPath),
	)

	ch := make(chan *file)

	// 监听通道
	wg := listen(ch)

	var s scan
	switch category {
	case "folder":
		s = newFolder(basePath)
	case "links":
		s = newLinks(basePath)
	default:
		log.Println("only two choices: links or folder")
	}

	s.scan(ch)

	wg.Wait()
}
