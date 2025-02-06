NAME=yaml2srs
BINDIR=build
GOBUILD=CGO_ENABLED=0 go build --ldflags="-s -w"
GOFILE=cmd/yaml2srs/main.go
EXAMPLE_PATH=examples
EXAMPLE_OUTPUT_PATH=${EXAMPLE_PATH}/output

EXAMPLE_FOLDER_PATH=${EXAMPLE_PATH}/folder


EXAMPLE_LINKS_PATH=${EXAMPLE_PATH}/links
EXAMPLE_LINKS_TXT_PATH=${EXAMPLE_LINKS_PATH}/links.txt


clean:
	rm -rf ${BINDIR} ${EXAMPLE_OUTPUT_PATH}

build: linux-amd64

linux-amd64: 
	GOARCH=amd64 GOOS=linux $(GOBUILD) -o $(BINDIR)/$(NAME)-$@ $(GOFILE)

folder: 
	$(BINDIR)/$(NAME)-linux-amd64 folder -p ${EXAMPLE_FOLDER_PATH} -o ${EXAMPLE_OUTPUT_PATH} 

links: 
	$(BINDIR)/$(NAME)-linux-amd64 links -p ${EXAMPLE_LINKS_TXT_PATH} -o ${EXAMPLE_OUTPUT_PATH}


