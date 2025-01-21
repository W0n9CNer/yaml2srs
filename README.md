# yaml2srs

#### 灵感来源: 
https://github.com/SagerNet/sing-box  
https://github.com/Toperlock/sing-box-geosite  

#### 用法:
1. 安装yaml2srs
```
go install github.com/W0n9CNer/yaml2srs/cmd/yaml2srs@latest
```
2. 命令
```
Usage:
  yaml2srs [command]

Available Commands:
  completion  Generate the autocompletion script for the specified shell
  folder      Specify the path of the folder
  help        Help about any command
  links       Specify the path of the links.txt file
  version     Print the Version number of yaml2srs

Flags:
  -h, --help   help for yaml2srs
```
- folder: 采用指定规则目录的方式,遍历规则目录中的yaml格式规则文件,将其转换成srs二进制规则集  
  - p: 指定规则目录 (-p examples/folder)  
  - o: json规则集和srs二进制规则集的输出目录 (-o examples/output)
```
Usage:
  yaml2srs folder [flags]

Flags:
  -h, --help                help for folder
  -o, --outputPath string   json file and srs file output path
  -p, --path string         links or folder path

```
```
yaml2srs folder -p examples/folder -o examples/output
```

- links: 采用读取文件的方式,解析文件中的yaml格式规则链接,将其转换成srs二进制规则集
  - p: 指定规则目录 (-p examples/links/links.txt)  
  - o: json规则集和srs二进制规则集的输出目录 (-o examples/output)
```
Usage:
  yaml2srs links [flags]

Flags:
  -h, --help                help for links
  -o, --outputPath string   json file and srs file output path
  -p, --path string         links or folder path
```
```
yaml2srs links -p examples/links/links.txt -o examples/output
```
