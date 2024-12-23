# IP地理信息查询API

这是一个基于Go语言开发的IP地理信息查询API，使用MaxMind的GeoIP2数据库来提供IP地址的详细信息。

## 项目结构

```
ip-geo/
├── cmd/                    # 可执行文件目录
│   └── server/            # 服务器入口
│       └── main.go
├── internal/              # 内部包
│   ├── api/              # API相关代码
│   │   ├── handler/      # HTTP处理器
│   │   └── response/     # 响应模型
│   ├── config/           # 配置管理
│   ├── database/         # 数据库连接管理
│   └── service/          # 业务逻辑
├── pkg/                   # 公共包
│   └── asn/              # ASN映射
├── config.json           # 配置文件
└── README.md
```

## 功能特点

- 支持IPv4和IPv6地址查询
- 提供ASN（自治系统号码）信息
- 提供国家和地区信息
- 支持中文结果输出
- 包含主要中国运营商的ASN映射
- 支持配置文件
- 使用单例模式管理数据库连接
- 遵循Go项目最佳实践

## 依赖要求

- Go 1.22或更高版本
- MaxMind GeoIP2数据库文件：
  - GeoLite2-ASN.mmdb
  - GeoLite2-City.mmdb
  - GeoCN.mmdb

## 安装和运行

1. 克隆仓库：
```bash
git clone [repository-url]
cd ip-geo
```

2. 安装依赖：
```bash
go mod tidy
```

3. 确保MaxMind数据库文件存在于项目根目录：
   - GeoLite2-ASN.mmdb
   - GeoLite2-City.mmdb
   - GeoCN.mmdb

4. 配置服务器（可选）：
   编辑 `config.json` 文件：
```json
{
    "server": {
        "port": 8080
    },
    "database": {
        "asn_db_path": "GeoLite2-ASN.mmdb",
        "city_db_path": "GeoLite2-City.mmdb",
        "geo_cn_db_path": "GeoCN.mmdb"
    }
}
```

5. 运行服务器：
```bash
go run cmd/server/main.go
```

服务器将在配置的端口上启动（默认8080）。

## API端点

### 1. 获取当前IP信息

```
GET /api/ip/current
```

返回发起请求的IP地址的详细信息。

### 2. 查询指定IP信息

```
GET /api/ip/query/{ip}
```

参数：
- `ip`: 要查询的IP地址（支持IPv4和IPv6）

## 响应示例

```json
{
    "ip": "2409:8a00:1:0:0:0:0:1a2b",
    "as.number": 56048,
    "as.name": "China Mobile Communications Corporation",
    "as.info": "中国移动",
    "addr": "2409:8a00::/37",
    "country.code": "CN",
    "country.name": "中国",
    "registered_country.code": "CN",
    "registered_country.name": "中国",
    "regions": ["北京市", "东城区"],
    "regions_short": ["北京", "东城区"],
    "type": "宽带"
}
```

## 错误处理

- 400 Bad Request: 无效的IP地址
- 500 Internal Server Error: 服务器内部错误

## 开发

### 项目结构说明

- `cmd/server/main.go`: 主程序入口
- `internal/api/handler/`: HTTP请求处理器
- `internal/api/response/`: API响应模型
- `internal/config/`: 配置管理
- `internal/database/`: 数据库连接管理
- `internal/service/`: 业务逻辑实现
- `pkg/asn/`: ASN映射数据

### 添加新功能

1. 在适当的包中添加新代码
2. 更新测试
3. 更新文档
4. 提交代码

## 许可证

MIT License 