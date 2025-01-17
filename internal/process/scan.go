package process

type scan interface {
	scan(ch chan *file)
}

type basePath struct {
	path       string // links Path or folder Path
	outputPath string // json file And srs outputPath
}

type pathOption func(s *basePath)

func withPath(p string) pathOption {
	return func(s *basePath) {
		s.path = p
	}
}
func withOutputPath(outputPath string) pathOption {
	return func(s *basePath) {
		s.outputPath = outputPath
	}
}

func newBasePath(options ...pathOption) *basePath {
	s := new(basePath)
	for _, o := range options {
		o(s)
	}
	return s
}

type file struct {
	fullFileName string // abc.yaml
	fileName     string // abc
	fileSuffix   string // .yaml
	fileBody     []byte // 未格式化的原始数据
	filePath     *basePath
}
type fileOption func(s *file)

func withFullFileName(fullFileName string) fileOption {
	return func(f *file) {
		f.fullFileName = fullFileName
	}
}
func withFileName(fileName string) fileOption {
	return func(f *file) {
		f.fileName = fileName
	}
}

func withFileSuffix(fileSuffix string) fileOption {
	return func(f *file) {
		f.fileSuffix = fileSuffix
	}
}

func withFileBody(body []byte) fileOption {
	return func(f *file) {
		f.fileBody = body
	}
}
func withFileOutputPath(path *basePath) fileOption {
	return func(f *file) {
		f.filePath = path
	}
}
func newFile(options ...fileOption) *file {
	s := new(file)
	for _, o := range options {
		o(s)
	}
	return s
}
