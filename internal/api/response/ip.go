package response

// IPResponse 表示IP查询的响应结构
type IPResponse struct {
	IP      string `json:"ip"`
	Version string `json:"version"`
	ASN     struct {
		Number uint   `json:"number"`
		Name   string `json:"name"`
		Info   string `json:"info"`
	} `json:"asn"`
	Network struct {
		CIDR     string `json:"cidr"`
		StartIP  string `json:"start_ip"`
		EndIP    string `json:"end_ip"`
		TotalIPs uint64 `json:"total_ips"`
		Type     string `json:"type"`
	} `json:"network"`
	Location struct {
		Country struct {
			Code string `json:"code"`
			Name string `json:"name"`
		} `json:"country"`
		Region   Region `json:"region"`
		Timezone string `json:"timezone"`
	} `json:"location"`
	ISP struct {
		Name string `json:"name"`
		Type string `json:"type"`
	} `json:"isp"`
}

// Region 表示地区信息
type Region struct {
	Code string `json:"code"`
	Name string `json:"name"`
}
