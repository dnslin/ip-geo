package service

import (
	"fmt"
	"math"
	"net"
	"strings"

	"ip-geo/internal/api/response"
	"ip-geo/internal/database"
	"ip-geo/internal/logger"
	"ip-geo/pkg/asn"
)

// IPService 处理IP查询相关的业务逻辑
type IPService struct {
	db *database.MMDBManager
}

// NewIPService 创建新的IPService实例
func NewIPService() *IPService {
	return &IPService{
		db: database.GetInstance(),
	}
}

// LookupIP 查询IP信息
func (s *IPService) LookupIP(ip string) (*response.IPResponse, error) {
	logger.Info("开始查询IP: %s", ip)
	resp := &response.IPResponse{IP: ip}

	// 解析IP地址
	parsedIP := net.ParseIP(ip)
	if parsedIP == nil {
		logger.Warn("无效的IP地址: %s", ip)
		return nil, ErrInvalidIP
	}

	// 设置IP版本
	if parsedIP.To4() != nil {
		resp.Version = "IPv4"
	} else {
		resp.Version = "IPv6"
	}
	logger.Debug("IP版本: %s", resp.Version)

	// 查询ASN信息
	if err := s.lookupASN(parsedIP, resp); err != nil {
		logger.Warn("查询ASN信息失败: %v", err)
	}

	// 查询地理位置信息
	// 先尝试从GeoCN数据库获取中国IP信息
	if err := s.lookupGeoCN(parsedIP, resp); err != nil {
		logger.Debug("从GeoCN查询失败，尝试使用GeoIP2: %v", err)
		// 如果GeoCN查询失败，使用GeoIP2数据库
		if err := s.lookupGeoIP2(parsedIP, resp); err != nil {
			logger.Warn("GeoIP2查询也失败: %v", err)
		}
	}

	// 如果网络信息仍然为空，设置默认网段
	if resp.Network.CIDR == "" {
		s.setDefaultNetwork(parsedIP, resp)
	}

	logger.Info("IP查询完成: %s", ip)
	return resp, nil
}

// lookupASN 查询ASN信息
func (s *IPService) lookupASN(ip net.IP, resp *response.IPResponse) error {
	var asnRecord struct {
		AutonomousSystemNumber       uint      `maxminddb:"autonomous_system_number"`
		AutonomousSystemOrganization string    `maxminddb:"autonomous_system_organization"`
		Network                      net.IPNet `maxminddb:"network"`
	}

	if err := s.db.ASNDB.Lookup(ip, &asnRecord); err != nil {
		return err
	}

	resp.ASN.Number = asnRecord.AutonomousSystemNumber
	resp.ASN.Name = asnRecord.AutonomousSystemOrganization
	if info, ok := asn.Map[int(asnRecord.AutonomousSystemNumber)]; ok {
		resp.ASN.Info = info
		resp.ISP.Type = info
	}
	resp.ISP.Name = asnRecord.AutonomousSystemOrganization

	if asnRecord.Network.IP != nil && asnRecord.Network.Mask != nil {
		s.setNetworkInfo(resp, asnRecord.Network)
	}

	return nil
}

// lookupGeoCN 从GeoCN数据库查询信息
func (s *IPService) lookupGeoCN(ip net.IP, resp *response.IPResponse) error {
	var geoCNRecord struct {
		Province      string `maxminddb:"province"`
		ProvinceCode  uint64 `maxminddb:"provinceCode"`
		City          string `maxminddb:"city"`
		CityCode      uint64 `maxminddb:"cityCode"`
		Districts     string `maxminddb:"districts"`
		DistrictsCode uint64 `maxminddb:"districtsCode"`
		ISP           string `maxminddb:"isp"`
		Net           string `maxminddb:"net"`
	}

	if err := s.db.GeoCNDB.Lookup(ip, &geoCNRecord); err != nil {
		return err
	}

	// 只有当Province或ISP字段不为空时才认为是有效的中国IP记录
	if geoCNRecord.Province == "" && geoCNRecord.ISP == "" {
		return fmt.Errorf("无效的GeoCN记录")
	}

	// 设置基本信息
	resp.Location.Country.Code = "CN"
	resp.Location.Country.Name = "中国"
	resp.Location.Location.TimeZone = "Asia/Shanghai"

	// 处理地区信息
	regions := []string{geoCNRecord.Province, geoCNRecord.City, geoCNRecord.Districts}
	regions = removeEmpty(regions)
	fullName := strings.Join(regions, "")

	// 设置地区代码
	var lastCode string
	if geoCNRecord.Districts != "" {
		lastCode = fmt.Sprintf("%d", geoCNRecord.DistrictsCode)
	} else if geoCNRecord.City != "" {
		lastCode = fmt.Sprintf("%d", geoCNRecord.CityCode)
	} else if geoCNRecord.Province != "" {
		lastCode = fmt.Sprintf("%d", geoCNRecord.ProvinceCode)
	}

	resp.Location.Region = response.Region{
		Code: lastCode,
		Name: fullName,
	}

	// 设置ISP和网络信息
	if geoCNRecord.ISP != "" {
		resp.ISP.Name = geoCNRecord.ISP
		resp.ASN.Info = geoCNRecord.ISP
	}
	if geoCNRecord.Net != "" {
		resp.Network.Type = geoCNRecord.Net
	}

	// 从GeoIP2-City数据库补充经纬度等信息
	var cityRecord struct {
		Location struct {
			AccuracyRadius uint16  `maxminddb:"accuracy_radius"`
			Latitude       float64 `maxminddb:"latitude"`
			Longitude      float64 `maxminddb:"longitude"`
		} `maxminddb:"location"`
		Continent struct {
			Code  string            `maxminddb:"code"`
			Names map[string]string `maxminddb:"names"`
		} `maxminddb:"continent"`
	}

	if err := s.db.CityDB.Lookup(ip, &cityRecord); err == nil {
		// 补充位置信息
		resp.Location.Location.Latitude = cityRecord.Location.Latitude
		resp.Location.Location.Longitude = cityRecord.Location.Longitude
		resp.Location.Location.AccuracyRadius = cityRecord.Location.AccuracyRadius

		// 补充大洲信息
		if resp.Location.Continent.Code == "" {
			resp.Location.Continent.Code = cityRecord.Continent.Code
			resp.Location.Continent.Name = getLocalizedName(cityRecord.Continent.Names, "zh-CN", "en")
		}

		logger.Debug("从GeoIP2补充位置信息 - 经度: %f, 纬度: %f, 精度: %d",
			cityRecord.Location.Longitude,
			cityRecord.Location.Latitude,
			cityRecord.Location.AccuracyRadius)
	} else {
		logger.Debug("从GeoIP2补充位置信息失败: %v", err)
	}

	return nil
}

// lookupGeoIP2 从GeoIP2数据库查询信息
func (s *IPService) lookupGeoIP2(ip net.IP, resp *response.IPResponse) error {
	var record struct {
		Continent struct {
			Code      string            `maxminddb:"code"`
			GeonameID uint32            `maxminddb:"geoname_id"`
			Names     map[string]string `maxminddb:"names"`
		} `maxminddb:"continent"`
		Country struct {
			GeonameID uint32            `maxminddb:"geoname_id"`
			ISOCode   string            `maxminddb:"iso_code"`
			Names     map[string]string `maxminddb:"names"`
		} `maxminddb:"country"`
		RegisteredCountry struct {
			GeonameID uint32            `maxminddb:"geoname_id"`
			ISOCode   string            `maxminddb:"iso_code"`
			Names     map[string]string `maxminddb:"names"`
		} `maxminddb:"registered_country"`
		City struct {
			GeonameID uint32            `maxminddb:"geoname_id"`
			Names     map[string]string `maxminddb:"names"`
		} `maxminddb:"city"`
		Subdivisions []struct {
			GeonameID uint32            `maxminddb:"geoname_id"`
			ISOCode   string            `maxminddb:"iso_code"`
			Names     map[string]string `maxminddb:"names"`
		} `maxminddb:"subdivisions"`
		Location struct {
			AccuracyRadius uint16  `maxminddb:"accuracy_radius"`
			Latitude       float64 `maxminddb:"latitude"`
			Longitude      float64 `maxminddb:"longitude"`
			TimeZone       string  `maxminddb:"time_zone"`
		} `maxminddb:"location"`
		Traits struct {
			IsAnycast bool `maxminddb:"is_anycast"`
		} `maxminddb:"traits"`
		Network net.IPNet `maxminddb:"network"`
	}

	if err := s.db.CityDB.Lookup(ip, &record); err != nil {
		return err
	}

	// 设置大洲信息
	if record.Continent.Code != "" {
		resp.Location.Continent.Code = record.Continent.Code
		resp.Location.Continent.Name = getLocalizedName(record.Continent.Names, "zh-CN", "en")
	}

	// 对于Anycast IP，使用registered_country的信息
	if record.Traits.IsAnycast {
		logger.Debug("检测到Anycast IP: %s", ip)
		resp.Location.Country.Code = record.RegisteredCountry.ISOCode
		resp.Location.Country.Name = getLocalizedName(record.RegisteredCountry.Names, "zh-CN", "en")

		// Anycast IP通常不设置具体的地区和城市信息
		return nil
	}

	// 设置国家信息（非Anycast IP）
	if record.Country.ISOCode != "" {
		resp.Location.Country.Code = record.Country.ISOCode
		resp.Location.Country.Name = getLocalizedName(record.Country.Names, "zh-CN", "en")
	}

	// 设置地区信息
	if len(record.Subdivisions) > 0 {
		subdivision := record.Subdivisions[0]
		resp.Location.Region = response.Region{
			Code: subdivision.ISOCode,
			Name: getLocalizedName(subdivision.Names, "zh-CN", "en"),
		}
	}

	// 设置城市信息
	if cityName := getLocalizedName(record.City.Names, "zh-CN", "en"); cityName != "" {
		resp.Location.City = response.City{
			Name: cityName,
		}
	}

	// 设置位置信息
	if record.Location.TimeZone != "" {
		resp.Location.Location.Latitude = record.Location.Latitude
		resp.Location.Location.Longitude = record.Location.Longitude
		resp.Location.Location.AccuracyRadius = record.Location.AccuracyRadius
		resp.Location.Location.TimeZone = record.Location.TimeZone
	}

	// 如果网络信息为空，使用GeoIP2的网络信息
	if resp.Network.CIDR == "" && record.Network.IP != nil && record.Network.Mask != nil {
		s.setNetworkInfo(resp, record.Network)
	}

	return nil
}

// setNetworkInfo 设置网络信息
func (s *IPService) setNetworkInfo(resp *response.IPResponse, network net.IPNet) {
	resp.Network.CIDR = network.String()
	startIP, endIP := calculateNetworkRange(network)
	resp.Network.StartIP = startIP.String()
	resp.Network.EndIP = endIP.String()
	resp.Network.TotalIPs = calculateTotalIPs(network)
}

// setDefaultNetwork 设置默认网段
func (s *IPService) setDefaultNetwork(ip net.IP, resp *response.IPResponse) {
	var ipNet net.IPNet
	if ip.To4() != nil {
		ipNet = net.IPNet{
			IP:   ip.Mask(net.CIDRMask(24, 32)),
			Mask: net.CIDRMask(24, 32),
		}
	} else {
		ipNet = net.IPNet{
			IP:   ip.Mask(net.CIDRMask(64, 128)),
			Mask: net.CIDRMask(64, 128),
		}
	}
	s.setNetworkInfo(resp, ipNet)
}

// GetRealIP 获取真实IP地址
func GetRealIP(headers map[string]string, remoteAddr string) string {
	logger.Debug("获取真实IP地址, RemoteAddr: %s", remoteAddr)
	// 从X-Real-IP获取
	if ip := headers["X-Real-IP"]; ip != "" {
		logger.Debug("从X-Real-IP获取到IP: %s", ip)
		return ip
	}
	// 从X-Forwarded-For获取
	if ip := headers["X-Forwarded-For"]; ip != "" {
		ips := strings.Split(ip, ",")
		if len(ips) > 0 {
			realIP := strings.TrimSpace(ips[0])
			logger.Debug("从X-Forwarded-For获取到IP: %s", realIP)
			return realIP
		}
	}
	// 直接从请求地址获取
	ip, _, _ := net.SplitHostPort(remoteAddr)
	logger.Debug("从RemoteAddr获取到IP: %s", ip)
	return ip
}

// 获取本地化名称
func getLocalizedName(names map[string]string, primaryLang, fallbackLang string) string {
	if name, ok := names[primaryLang]; ok {
		return name
	}
	if name, ok := names[fallbackLang]; ok {
		return name
	}
	return ""
}

// 计算网段的起始和结束IP
func calculateNetworkRange(network net.IPNet) (net.IP, net.IP) {
	// 计算起始IP
	startIP := make(net.IP, len(network.IP))
	copy(startIP, network.IP)
	startIP = startIP.Mask(network.Mask)

	// 计算结束IP
	endIP := make(net.IP, len(network.IP))
	copy(endIP, startIP)
	for i := range endIP {
		endIP[i] |= ^network.Mask[i]
	}

	return startIP, endIP
}

// 计算网段包含的IP总数
func calculateTotalIPs(network net.IPNet) uint64 {
	prefixLen, _ := network.Mask.Size()
	if network.IP.To4() != nil {
		return uint64(math.Pow(2, float64(32-prefixLen)))
	}
	return uint64(math.Pow(2, float64(128-prefixLen)))
}

// 辅助函数：移除空字符串
func removeEmpty(arr []string) []string {
	result := make([]string, 0)
	for _, str := range arr {
		if str != "" {
			result = append(result, str)
		}
	}
	return result
}

// 辅助函数：移除重复项
func removeDuplicates(arr []string) []string {
	seen := make(map[string]bool)
	result := make([]string, 0)
	for _, str := range arr {
		if !seen[str] {
			seen[str] = true
			result = append(result, str)
		}
	}
	return result
}

// 添加这个辅助函数来确定地区类型
func determineRegionType(region string) string {
	// 检查是否是省级行政区
	for _, province := range asn.Provinces {
		if strings.Contains(region, province) {
			return "province"
		}
	}

	// 检查是否是城市
	if strings.HasSuffix(region, "市") {
		return "city"
	}

	// 检查是否是区县
	if strings.HasSuffix(region, "区") || strings.HasSuffix(region, "县") {
		return "district"
	}

	return "unknown"
}

// getRegionCode 获取地区代码，持中国和国际区域代码
func getRegionCode(subdivisions []struct {
	ISOCode    string            `maxminddb:"iso_code"`
	Names      map[string]string `maxminddb:"names"`
	GeoNameID  uint              `maxminddb:"geoname_id"`
	Confidence int               `maxminddb:"confidence"`
}) string {
	if len(subdivisions) > 0 && subdivisions[0].ISOCode != "" {
		return subdivisions[0].ISOCode
	}
	return ""
}
