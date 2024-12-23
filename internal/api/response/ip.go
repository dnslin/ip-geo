package response

// IPResponse 表示IP查询的响应结构
type IPResponse struct {
	// 基本信息
	IP      string `json:"ip"`
	Version string `json:"version"` // IPv4 或 IPv6

	// ASN信息
	ASN struct {
		Number uint   `json:"number"`
		Name   string `json:"name"`
		Info   string `json:"info"` // 中文名称，如：中国电信、中国移动等
	} `json:"asn"`

	// 网络信息
	Network struct {
		CIDR     string `json:"cidr"`      // CIDR格式的网络地址
		StartIP  string `json:"start_ip"`  // 网段起始IP
		EndIP    string `json:"end_ip"`    // 网段结束IP
		TotalIPs uint64 `json:"total_ips"` // IP总数
		Type     string `json:"type"`      // 网络类型（宽带、数据中心等）
	} `json:"network"`

	// 地理位置信息
	Location struct {
		// 国家和地区信息
		Country struct {
			Code string `json:"code"` // 国家代码，如：CN
			Name string `json:"name"` // 国家名称，如：中国
		} `json:"country"`

		// 行政区划信息（从大到小排列）
		Regions []struct {
			Code string `json:"code"` // 地区代码
			Name string `json:"name"` // 地区名称（省份、城市、区县等）
			Type string `json:"type"` // 地区类型（province省份, city城市, district区县）
		} `json:"regions"`

		// 经纬度和时区
		Coordinates struct {
			Latitude  float64 `json:"latitude"`  // 纬度
			Longitude float64 `json:"longitude"` // 经度
		} `json:"coordinates"`
		Timezone string `json:"timezone"` // 时区，如：Asia/Shanghai
	} `json:"location"`

	// 运营商信息
	ISP struct {
		Name string `json:"name"` // 运营商名称
		Type string `json:"type"` // 运营商类型（电信/联通/移动/云服务等）
	} `json:"isp"`
}
