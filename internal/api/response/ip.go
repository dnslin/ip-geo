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
		Continent struct {
			Code string `json:"code"`
			Name string `json:"name"`
		} `json:"continent"`
		Country struct {
			Code      string `json:"code"`
			Name      string `json:"name"`
			GeonameID uint32 `json:"geoname_id,omitempty"`
		} `json:"country"`
		Region   Region `json:"region"`
		City     City   `json:"city,omitempty"`
		Location struct {
			Latitude       float64 `json:"latitude,omitempty"`
			Longitude      float64 `json:"longitude,omitempty"`
			AccuracyRadius uint16  `json:"accuracy_radius,omitempty"`
			TimeZone       string  `json:"timezone,omitempty"`
		} `json:"location"`
	} `json:"location"`
	ISP struct {
		Name string `json:"name"`
		Type string `json:"type"`
	} `json:"isp"`
}

// Region 表示地区信息
type Region struct {
	Code      string `json:"code"`
	Name      string `json:"name"`
	GeonameID uint32 `json:"geoname_id,omitempty"`
}

// City 表示城市信息
type City struct {
	Name      string `json:"name"`
	GeonameID uint32 `json:"geoname_id,omitempty"`
}
