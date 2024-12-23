package service

import (
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
	var asnRecord struct {
		AutonomousSystemNumber       uint      `maxminddb:"autonomous_system_number"`
		AutonomousSystemOrganization string    `maxminddb:"autonomous_system_organization"`
		Network                      net.IPNet `maxminddb:"network"`
	}
	if err := s.db.ASNDB.Lookup(parsedIP, &asnRecord); err == nil {
		logger.Debug("ASN信息: AS%d (%s)", asnRecord.AutonomousSystemNumber, asnRecord.AutonomousSystemOrganization)
		resp.ASN.Number = asnRecord.AutonomousSystemNumber
		resp.ASN.Name = asnRecord.AutonomousSystemOrganization
		if info, ok := asn.Map[int(asnRecord.AutonomousSystemNumber)]; ok {
			resp.ASN.Info = info
			resp.ISP.Type = info
			logger.Debug("ASN映射信息: %s", info)
		}
		resp.ISP.Name = asnRecord.AutonomousSystemOrganization

		// 设置网络信息
		if asnRecord.Network.IP != nil && asnRecord.Network.Mask != nil {
			resp.Network.CIDR = asnRecord.Network.String()
			startIP, endIP := calculateNetworkRange(asnRecord.Network)
			resp.Network.StartIP = startIP.String()
			resp.Network.EndIP = endIP.String()
			resp.Network.TotalIPs = calculateTotalIPs(asnRecord.Network)
			logger.Debug("网络信息: %s (总IP数: %d)", resp.Network.CIDR, resp.Network.TotalIPs)
		}
	} else {
		logger.Warn("查询ASN信息失败: %v", err)
	}

	// 首先尝试从GeoCN数据库获取中国地区的详细信息
	var geoCNRecord struct {
		Country struct {
			ISOCode string            `maxminddb:"iso_code"`
			Names   map[string]string `maxminddb:"names"`
		} `maxminddb:"country"`
		Province  string `maxminddb:"province"`  // 省份
		City      string `maxminddb:"city"`      // 城市
		Districts string `maxminddb:"districts"` // 区县
		ISP       string `maxminddb:"isp"`       // ISP信息
		Net       string `maxminddb:"net"`       // 网络类型
		Location  struct {
			Latitude  float64 `maxminddb:"latitude"`
			Longitude float64 `maxminddb:"longitude"`
		} `maxminddb:"location"`
	}

	isChineseIP := false
	if err := s.db.GeoCNDB.Lookup(parsedIP, &geoCNRecord); err == nil && geoCNRecord.Country.ISOCode == "CN" {
		logger.Info("从GeoCN数据库获取到中国IP信息")
		isChineseIP = true
		// 设置国家信息
		resp.Location.Country.Code = "CN"
		resp.Location.Country.Name = "中国"
		resp.Location.Timezone = "Asia/Shanghai"

		// 设置经纬度
		if geoCNRecord.Location.Latitude != 0 || geoCNRecord.Location.Longitude != 0 {
			resp.Location.Coordinates.Latitude = geoCNRecord.Location.Latitude
			resp.Location.Coordinates.Longitude = geoCNRecord.Location.Longitude
		}

		// 设置ISP和网络类型信息
		if geoCNRecord.ISP != "" {
			resp.ISP.Name = geoCNRecord.ISP
			resp.ASN.Info = geoCNRecord.ISP
		}
		if geoCNRecord.Net != "" {
			resp.Network.Type = geoCNRecord.Net
		}

		// 处理地区信息（省市区）
		regions := []string{geoCNRecord.Province, geoCNRecord.City, geoCNRecord.Districts}
		regions = removeDuplicates(removeEmpty(regions)) // 去重和移除空值

		for i, region := range regions {
			var regionType string
			var regionName string

			switch i {
			case 0: // 省份
				regionType = "province"
				// 检查是否是特别行政区
				isSpecialRegion := false
				for _, special := range asn.SpecialRegions {
					if strings.Contains(region, special) {
						isSpecialRegion = true
						regionName = special
						break
					}
				}
				if !isSpecialRegion {
					// 尝试从省份映射中获取标准名称
					for shortName, fullName := range asn.ProvinceMap {
						if strings.Contains(region, shortName) {
							regionName = fullName
							break
						}
					}
				}
				if regionName == "" {
					regionName = region
				}
			case 1: // 城市
				regionType = "city"
				if !strings.HasSuffix(region, "市") {
					regionName = region + "市"
				} else {
					regionName = region
				}
			case 2: // 区县
				regionType = "district"
				if !strings.HasSuffix(region, "区") && !strings.HasSuffix(region, "县") {
					regionName = region + "区"
				} else {
					regionName = region
				}
			}

			if regionName != "" {
				resp.Location.Regions = append(resp.Location.Regions, struct {
					Code string `json:"code"`
					Name string `json:"name"`
					Type string `json:"type"`
				}{
					Code: "",
					Name: regionName,
					Type: regionType,
				})
			}
		}
	} else if err != nil {
		logger.Warn("查询GeoCN数据库失败: %v", err)
	}

	// 如果不是中国IP或者GeoCN数据库没有信息，则使用GeoLite2-City数据库
	if !isChineseIP {
		logger.Info("使用GeoLite2-City数据库查询非中国IP信息")
		var cityRecord struct {
			Country struct {
				ISOCode string            `maxminddb:"iso_code"`
				Names   map[string]string `maxminddb:"names"`
			} `maxminddb:"country"`

			City struct {
				Names     map[string]string `maxminddb:"names"`
				Latitude  float64           `maxminddb:"latitude"`
				Longitude float64           `maxminddb:"longitude"`
			} `maxminddb:"city"`
			Subdivisions []struct {
				ISOCode string            `maxminddb:"iso_code"`
				Names   map[string]string `maxminddb:"names"`
			} `maxminddb:"subdivisions"`
			Location struct {
				TimeZone string `maxminddb:"time_zone"`
			} `maxminddb:"location"`
			Network net.IPNet `maxminddb:"network"`
		}

		if err := s.db.CityDB.Lookup(parsedIP, &cityRecord); err == nil {
			// 设置国家信息
			resp.Location.Country.Code = cityRecord.Country.ISOCode
			resp.Location.Country.Name = getLocalizedName(cityRecord.Country.Names, "zh-CN", "en")

			// 设置地区信息
			for _, subdivision := range cityRecord.Subdivisions {
				resp.Location.Regions = append(resp.Location.Regions, struct {
					Code string `json:"code"`
					Name string `json:"name"`
					Type string `json:"type"`
				}{
					Code: subdivision.ISOCode,
					Name: getLocalizedName(subdivision.Names, "zh-CN", "en"),
					Type: "province",
				})
			}

			// 设置城市信息
			cityName := getLocalizedName(cityRecord.City.Names, "zh-CN", "en")
			if cityName != "" {
				resp.Location.Regions = append(resp.Location.Regions, struct {
					Code string `json:"code"`
					Name string `json:"name"`
					Type string `json:"type"`
				}{
					Code: "",
					Name: cityName,
					Type: "city",
				})
			}

			// 设置经纬度和时区
			resp.Location.Coordinates.Latitude = cityRecord.City.Latitude
			resp.Location.Coordinates.Longitude = cityRecord.City.Longitude
			resp.Location.Timezone = cityRecord.Location.TimeZone

			// 如果ASN查询没有获取到网络信息，使用城市数据库的网络信息
			if resp.Network.CIDR == "" && cityRecord.Network.IP != nil && cityRecord.Network.Mask != nil {
				resp.Network.CIDR = cityRecord.Network.String()
				startIP, endIP := calculateNetworkRange(cityRecord.Network)
				resp.Network.StartIP = startIP.String()
				resp.Network.EndIP = endIP.String()
				resp.Network.TotalIPs = calculateTotalIPs(cityRecord.Network)
			}
		} else {
			logger.Warn("查询City数据库失败: %v", err)
		}
	}

	// 设置网络类型
	if resp.Network.Type == "" {
		resp.Network.Type = "宽带" // 默认网络类型
	}

	logger.Info("IP查询完成: %s", ip)
	return resp, nil
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
	startIP := network.IP.Mask(network.Mask)
	endIP := make(net.IP, len(startIP))
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
