package handler

import (
	"encoding/json"
	"net"
	"net/http"
	"strings"

	"ip-geo/internal/logger"
	"ip-geo/internal/service"
)

// IPHandler 处理IP相关的HTTP请求
type IPHandler struct {
	ipService *service.IPService
}

// NewIPHandler 创建新的IPHandler实例
func NewIPHandler() *IPHandler {
	return &IPHandler{
		ipService: service.NewIPService(),
	}
}

// HandleCurrentIP 处理当前IP查询请求
func (h *IPHandler) HandleCurrentIP(w http.ResponseWriter, r *http.Request) {
	// 添加CORS头
	h.setCORSHeaders(w)

	// 处理预检请求
	if r.Method == "OPTIONS" {
		return
	}

	// 按优先级获取真实IP
	ip := h.getRealIPFromRequest(r)
	logger.Debug("获取到客户端IP: %s", ip)
	h.handleIPLookup(w, ip)
}

// getRealIPFromRequest 按优先级从请求中获取真实IP地址
func (h *IPHandler) getRealIPFromRequest(r *http.Request) string {

	// 2. 从X-Real-IP获取
	if ip := r.Header.Get("X-Real-IP"); ip != "" {
		logger.Debug("从X-Real-IP获取到IP: %s", ip)
		return ip
	}

	// 3. 从X-Forwarded-For获取第一个IP
	if forwardedFor := r.Header.Get("X-Forwarded-For"); forwardedFor != "" {
		ips := strings.Split(forwardedFor, ",")
		if len(ips) > 0 {
			ip := strings.TrimSpace(ips[0])
			logger.Debug("从X-Forwarded-For获取到IP: %s", ip)
			return ip
		}
	}

	// 4. 从RemoteAddr获取
	ip, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		// 如果分割失败,说明可能没有端口号,直接使用完整地址
		logger.Debug("从RemoteAddr获取到IP(无端口): %s", r.RemoteAddr)
		return r.RemoteAddr
	}
	// 1. 从Cloudflare的CF-Connecting-IP获取
	if ip := r.Header.Get("CF-Connecting-IP"); ip != "" {
		logger.Debug("从CF-Connecting-IP获取到IP: %s", ip)
		return ip
	}

	logger.Debug("从RemoteAddr获取到IP: %s", ip)
	return ip
}

// HandleQueryIP 处理指定IP查询请求
func (h *IPHandler) HandleQueryIP(w http.ResponseWriter, r *http.Request) {
	// 添加CORS头
	h.setCORSHeaders(w)

	// 处理预检请求
	if r.Method == "OPTIONS" {
		return
	}

	ip := r.PathValue("ip")
	h.handleIPLookup(w, ip)
}

// setCORSHeaders 设置CORS响应头
func (h *IPHandler) setCORSHeaders(w http.ResponseWriter) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type, X-Real-IP, X-Forwarded-For, CF-Connecting-IP")
	w.Header().Set("Access-Control-Max-Age", "3600")
}

// handleIPLookup 处理IP查询
func (h *IPHandler) handleIPLookup(w http.ResponseWriter, ip string) {
	response, err := h.ipService.LookupIP(ip)
	if err != nil {
		if err == service.ErrInvalidIP {
			http.Error(w, err.Error(), http.StatusBadRequest)
		} else {
			logger.Error("查询IP信息失败: %v", err)
			http.Error(w, "服务器内部错误", http.StatusInternalServerError)
		}
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		logger.Error("编码响应失败: %v", err)
		http.Error(w, "服务器内部错误", http.StatusInternalServerError)
		return
	}
}
