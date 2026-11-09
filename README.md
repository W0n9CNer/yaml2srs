# yaml2srs

#### 灵感来源:
https://github.com/SagerNet/sing-box  
https://github.com/Toperlock/sing-box-geosite  

把 Clash / mihomo 格式的 yaml 规则(rule provider)转换成 sing-box 的 json 规则集和 srs 二进制规则集。

#### 用法:
1. 安装yaml2srs
```
go install github.com/W0n9CNer/yaml2srs/cmd/yaml2srs@latest
```
或者从源码构建:
```bash
git clone https://github.com/W0n9CNer/yaml2srs
cd yaml2srs
make build            # 产物在 build/yaml2srs-linux-amd64
# 或
go build -o yaml2srs ./cmd/yaml2srs
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

#### 支持的规则类型

| Clash | sing-box |
| --- | --- |
| DOMAIN | domain |
| DOMAIN-SUFFIX | domain_suffix |
| DOMAIN-KEYWORD | domain_keyword |
| DOMAIN-REGEX | domain_regex |
| IP-CIDR / IP-CIDR6 | ip_cidr |
| 裸域名 / `.example.com` / `+.example.com` / `*.example.com` 通配写法 | domain / domain_suffix / domain_regex |

`PROCESS-NAME`、`IP-ASN`、`GEOIP` 等无法在规则集里表达的类型会被跳过,并在日志里按文件汇总打印
`unsupported rule type ...`,方便发现上游新增的规则类型。

#### 行为说明

- 跳过文件名含 `Resolve` 的规则文件(no-resolve 变体),只转换常规版本。
- 上游部分规则的 `<名称>.yaml` 只保留了少量条目,完整列表在 `<名称>_Classical.yaml` 里
  (例如 ChinaMax 只有 27 条 vs Classical 的 123062 条)。转换时会自动把同目录下的
  `_Classical.yaml` 合并进来,避免生成几乎为空的规则集。
- 上游存在同名规则文件时(例如 `Direct/Direct.yaml` 与 `AdGuardSDNSFilter/Direct/Direct.yaml`),
  按路径深度、再按字典序取第一个,保证每次生成结果一致。
- 生成的 srs 使用二进制格式版本 1,兼容旧版 sing-box;json 与 srs 先写临时文件再重命名,
  单个文件失败不会留下半个文件。
- 依赖 `github.com/sagernet/sing-box` 的规则集编解码库,`yaml2srs version` 会打印当前编译进去的 sing-box 版本。

## License

MIT
