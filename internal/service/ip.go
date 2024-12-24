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
		Province      string `maxminddb:"province"`
		ProvinceCode  uint64 `maxminddb:"provinceCode"`
		City          string `maxminddb:"city"`
		CityCode      uint64 `maxminddb:"cityCode"`
		Districts     string `maxminddb:"districts"`
		DistrictsCode uint64 `maxminddb:"districtsCode"`
		ISP           string `maxminddb:"isp"`
		Net           string `maxminddb:"net"`
	}

	// 先尝试从GeoCN数据库查询
	err := s.db.GeoCNDB.Lookup(parsedIP, &geoCNRecord)

	// // 清理字符串字段
	// geoCNRecord.Province = strings.TrimSpace(geoCNRecord.Province)
	// geoCNRecord.City = strings.TrimSpace(geoCNRecord.City)
	// geoCNRecord.Districts = strings.TrimSpace(geoCNRecord.Districts)
	// geoCNRecord.ISP = strings.TrimSpace(geoCNRecord.ISP)
	// geoCNRecord.Net = strings.TrimSpace(geoCNRecord.Net)

	// logger.Debug("GeoCN完整结构: %+v", geoCNRecord)
	// logger.Debug("GeoCN查询错误: %v", err)
	logger.Debug("resp.Network.CIDR: %s", resp.Network.CIDR)
	// 如果网络信息为空，根据IP类型设置默认网段
	if resp.Network.CIDR == "" {
		var ipNet net.IPNet
		if parsedIP.To4() != nil {
			// IPv4 使用 /24 网段
			ipNet = net.IPNet{
				IP:   parsedIP.Mask(net.CIDRMask(24, 32)), // 将IP掩码为/24
				Mask: net.CIDRMask(24, 32),
			}
		} else {
			// IPv6 使用 /64 网段
			ipNet = net.IPNet{
				IP:   parsedIP.Mask(net.CIDRMask(64, 128)), // 将IP掩码为/64
				Mask: net.CIDRMask(64, 128),
			}
		}
		
		resp.Network.CIDR = ipNet.String()
		startIP, endIP := calculateNetworkRange(ipNet)
		resp.Network.StartIP = startIP.String()
		resp.Network.EndIP = endIP.String()
		resp.Network.TotalIPs = calculateTotalIPs(ipNet)
		logger.Debug("使用默认网段: %s (总IP数: %d)", resp.Network.CIDR, resp.Network.TotalIPs)
	}
	// 通过检查Province或ISP字段来判断是否为中国IP
	if err == nil && (geoCNRecord.Province != "" || geoCNRecord.ISP != "") {
		logger.Info("从GeoCN数据库获取到中国IP信息")
		
		// 设置国家信息
		resp.Location.Country.Code = "CN"
		resp.Location.Country.Name = "中国"
		resp.Location.Timezone = "Asia/Shanghai"

		// 处理地区信息
		regions := []string{geoCNRecord.Province, geoCNRecord.City, geoCNRecord.Districts}
		regions = removeEmpty(regions) // 只需要移除空值，不需要去重

		// 记录日志以便调试
		logger.Debug("原始地区信息: Province=%s, City=%s, Districts=%s", 
			geoCNRecord.Province, geoCNRecord.City, geoCNRecord.Districts)
		logger.Debug("处理后的地区信息: %v", regions)

		// 构建完整地址名称和最后一级代码
		fullName := strings.Join(regions, "")
		var lastCode string
		if geoCNRecord.Districts != "" {
			lastCode = fmt.Sprintf("%d", geoCNRecord.DistrictsCode)
		} else if geoCNRecord.City != "" {
			lastCode = fmt.Sprintf("%d", geoCNRecord.CityCode)
		} else if geoCNRecord.Province != "" {
			lastCode = fmt.Sprintf("%d", geoCNRecord.ProvinceCode)
		}

		// 设置地区信息
		resp.Location.Region = response.Region{
			Code: lastCode,
			Name: fullName,
		}

		// 设置ISP信息
		if geoCNRecord.ISP != "" {
			resp.ISP.Name = geoCNRecord.ISP
			resp.ASN.Info = geoCNRecord.ISP  // 同时更新ASN信息
		}
		if geoCNRecord.Net != "" {
			resp.Network.Type = geoCNRecord.Net
		}

		return resp, nil
	} else {
		logger.Info("未进入中国IP分支的原因: err=%v, Province=[%s], ISP=[%s]", 
			err, geoCNRecord.Province, geoCNRecord.ISP)
		// 如果不是中国IP或者GeoCN数据库查询失败则使用GeoLite2-City数据库
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

			// 收集所有地区名称
			var regions []string

			// 添加省份/州信息
			for _, subdivision := range cityRecord.Subdivisions {
				regions = append(regions, getLocalizedName(subdivision.Names, "zh-CN", "en"))
			}

			// 添加城市信息
			cityName := getLocalizedName(cityRecord.City.Names, "zh-CN", "en")
			if cityName != "" {
				regions = append(regions, cityName)
			}

			// 设置地区信息
			resp.Location.Region = response.Region{
				Code: getRegionCode(cityRecord.Subdivisions),
				Name: strings.Join(regions, ""),
			}

			// 设置经纬度和时区
			resp.Location.Timezone = cityRecord.Location.TimeZone
			// 如果ASN查询没有获取到网络信息，使用城市数据库的网络信息
			if resp.Network.CIDR == "" && cityRecord.Network.IP != nil && cityRecord.Network.Mask != nil {
				resp.Network.CIDR = cityRecord.Network.String()
				startIP, endIP := calculateNetworkRange(cityRecord.Network)
				resp.Network.StartIP = startIP.String()
				resp.Network.EndIP = endIP.String()
				resp.Network.TotalIPs = calculateTotalIPs(cityRecord.Network)
				logger.Debug("从City数据库获取网络信息: %s (总IP数: %d)", resp.Network.CIDR, resp.Network.TotalIPs)
			}
		} else {
			logger.Warn("查询City数据库失败: %v", err)
		}
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

// getRegionCode 获取地区代码，支持中国和国际区域代码
func getRegionCode(subdivisions []struct {
	ISOCode string            `maxminddb:"iso_code"`
	Names   map[string]string `maxminddb:"names"`
}) string {
	if len(subdivisions) > 0 && subdivisions[0].ISOCode != "" {
		return subdivisions[0].ISOCode
	}
	return ""
}
