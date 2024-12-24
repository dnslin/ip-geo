# IP-Geo API 服务

这是一个基于Go语言开发的IP地理位置查询服务，支持IPv4和IPv6地址的查询，能够提供IP地址的地理位置、网络、ASN等详细信息。

## 功能特性

- 支持IPv4和IPv6地址查询
- 提供详细的地理位置信息（国家、省份、城市等）
- 支持中国IP地址的精确定位
- 提供ASN（自治系统编号）信息
- 网络信息查询（CIDR、IP范围等）
- ISP（互联网服务提供商）信息
- 支持X-Real-IP和X-Forwarded-For的真实IP识别

## 项目结构

```
.
├── cmd/
│   └── server/           # 服务器入口
├── internal/
│   ├── api/             # API相关代码
│   │   ├── handler/     # 请求处理器
│   │   └── response/    # 响应结构定义
│   ├── service/         # 业务逻辑层
│   ├── database/        # 数据库管理
│   ├── logger/          # 日志管理
│   └── config/          # 配置管理
└── logs/                # 日志文件
```

## API接口

### 1. 获取当前IP信息

```
GET /ip
```

获取当前访问者的IP地址信息。

### 2. 查询指定IP信息

```
GET /ip/{ip}
```

查询指定IP地址的详细信息。

#### 响应示例

```json
{
  "ip": "8.8.8.8",
  "version": "IPv4",
  "asn": {
    "number": 15169,
    "name": "Google LLC",
    "info": "Google全球网络"
  },
  "location": {
    "country": {
      "code": "US",
      "name": "美国"
    },
    "region": {
      "code": "CA",
      "name": "加利福尼亚州"
    },
    "timezone": "America/Los_Angeles"
  },
  "network": {
    "cidr": "8.8.8.0/24",
    "startIP": "8.8.8.0",
    "endIP": "8.8.8.255",
    "totalIPs": 256,
    "type": "公网"
  },
  "isp": {
    "name": "Google LLC",
    "type": "全球网络"
  }
}
```

## 运行服务

1. 确保已安装Go 1.22或更高版本
2. 启动服务：

```bash
go run cmd/server/main.go
```

服务默认运行在`:8080`端口。

## 特性说明

- 使用Go 1.22新特性的ServeMux进行路由处理
- 支持多级代理的真实IP识别
- 完善的日志记录系统
- 高性能的IP地址解析和查询
- 支持多个IP数据库的联合查询
- 优雅的错误处理机制

## 依赖说明

- Go 1.22+
- MaxMind GeoIP2数据库
- 自定义中国IP数据库（GeoCN）

## 日志

日志文件存储在`logs`目录下，按日期命名（如：`2024-12-24.log`）。