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
  - o: 规则集输出目录 (-o examples/output)
```
Usage:
  yaml2srs folder [flags]

Flags:
  -h, --help                help for folder
  -o, --outputPath string   rule set output directory
  -p, --path string         links or folder path

```
```
yaml2srs folder -p examples/folder -o examples/output
```

- links: 采用读取文件的方式,解析文件中的yaml格式规则链接,将其转换成srs二进制规则集
  - p: 指定规则目录 (-p examples/links/links.txt)  
  - o: 规则集输出目录 (-o examples/output)
```
Usage:
  yaml2srs links [flags]

Flags:
  -h, --help                help for links
  -o, --outputPath string   rule set output directory
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
| IP-SUFFIX | ip_cidr |
| SRC-IP-CIDR / SRC-IP-SUFFIX | source_ip_cidr |
| DST-PORT | port / port_range |
| SRC-PORT | source_port / source_port_range |
| PROCESS-NAME | process_name |
| PROCESS-PATH | process_path |
| PROCESS-PATH-WILDCARD | process_path_regex |
| PROCESS-PATH-REGEX | process_path_regex |
| 裸域名 / `.example.com` / `+.example.com` / `*.example.com` 通配写法 | domain / domain_suffix / domain_regex |

`IP-ASN`、`GEOIP` 等无法在规则集里表达的类型会被跳过,并在日志里按文件汇总打印
`unsupported rule type ...`,方便发现上游新增的规则类型。`DOMAIN-WILDCARD` 不做映射
(通配语义与 sing-box 的 `domain_regex` 不完全一致),同样按 unsupported 跳过。

#### 行为说明

- 同时支持 `.yaml` 和 `.yml`;跳过以 `_Resolve.yaml` / `_No_Resolve.yaml` 结尾的变体
  (Clash fake-ip 专用,与普通版本内容重复),只转换常规版本。
- 上游部分规则的 `<名称>.yaml` 只保留了少量条目,完整列表在 `<名称>_Classical.yaml` 里
  (例如 ChinaMax 只有 27 条 vs Classical 的 123062 条)。转换时会自动把同目录下的
  `_Classical.yaml` 合并进来,避免生成几乎为空的规则集。
- 上游存在同名规则文件时(例如 `Direct/Direct.yaml` 与 `AdGuardSDNSFilter/Direct/Direct.yaml`),
  按路径深度、再按字典序取第一个,保证每次生成结果一致。`links` 模式按输出文件名去重,
  同名链接只下载第一个,避免并发写同一个输出文件。
- 域名会统一转成小写并去掉结尾的点;裸 IP(例如 `1.2.3.4`)以及 `DOMAIN,1.2.3.4` 这类
  上游把 IP 写成域名的条目会归到 `ip_cidr`,否则 sing-box 的 domain 规则永远匹配不到 IP。
- `IP-SUFFIX` / `SRC-IP-SUFFIX` 会先按前缀掩码规范化(例如 `8.8.8.8/24` → `8.8.8.0/24`);
  `DST-PORT` / `SRC-PORT` 支持单个端口、`-` 区间以及 `/`、`,` 分隔的多端口列表,
  单个端口写入 `port`/`source_port`,区间写入 `port_range`/`source_port_range`。
  `PROCESS-PATH-WILDCARD` 会转成带 `^...$` 锚点的 `process_path_regex`。
- `*.example.com` 会生成带 `^...$` 锚点、按标签转义的正则,避免误匹配 `evilwww.example.com`。
- Clash 规则集里每一行都是 OR 关系,而 sing-box 会在同一条 rule 内把不同条件组 AND 起来。
  因此同一份规则里同时出现域名、IP、端口、进程等不同类型时,会拆成多条 rule
  (域名与 `ip_cidr` 属于同一目的地址组,仍放在同一条 rule 内),避免把 Clash 的
  "任一命中"错误转换成 "全部命中"。
- 只有 `IP-ASN`/`GEOIP` 等无法表达条目的文件不会生成空规则集,而是跳过并在日志中提示。
  解析/下载/输出失败会让进程以非 0 退出,便于 CI 发现问题。
- 生成的 srs 使用二进制格式版本 1,兼容旧版 sing-box;所有文件先写临时文件再重命名,
  单个文件失败不会留下半个文件。
- 每个规则集输出五个文件:
  - `<规则名>.json`:带缩进的 json,便于查看和调试;
  - `<规则名>.min.json`:未格式化的紧凑 json,体积更小;
  - `<规则名>.srs`:sing-box 二进制规则集;
  - `<规则名>.srs.md5`:srs 的 md5,内容为 `md5sum` 格式,在文件所在目录执行
    `md5sum -c <规则名>.srs.md5` 即可校验;
  - `<规则名>.yaml`(或 `.yml`):生成该规则集所用的原始 yaml,按原字节复制,
    方便核对上游内容和复现问题。合并了 `_Classical` 的规则集附带的是主文件,
    `_Classical` 文件本身仍会作为独立规则集输出。
- 目录布局为 `<输出目录>/<规则名>/<规则名>.*`。
- 依赖 `github.com/sagernet/sing-box` 的规则集编解码库,`yaml2srs version` 会打印当前编译进去的 sing-box 版本。

#### 开发

```bash
make test   # go test ./...
make vet    # go vet ./...
make fmt    # gofmt
```

生成完整目录后可以做一次集成校验(会重新读取每个 json/srs 并交给 sing-box 解码,
同时确认每个目录都带有可解析的原始 yaml):

```bash
YAML2SRS_STAGE=/path/to/output go test ./internal/outputor -run TestValidateGeneratedDirectory -v
```

## License

MIT
