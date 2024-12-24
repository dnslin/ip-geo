package handler

import (
	"encoding/json"
	"net/http"
	"net"
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
	// 按优先级获取真实IP
	ip := h.getRealIPFromRequest(r)
	logger.Debug("获取到客户端IP: %s", ip)
	h.handleIPLookup(w, ip)
}

// getRealIPFromRequest 按优先级从请求中获取真实IP地址
func (h *IPHandler) getRealIPFromRequest(r *http.Request) string {
	// 1. 从X-Real-IP获取
	if ip := r.Header.Get("X-Real-IP"); ip != "" {
		logger.Debug("从X-Real-IP获取到IP: %s", ip)
		return ip
	}

	// 2. 从X-Forwarded-For获取第一个IP
	if forwardedFor := r.Header.Get("X-Forwarded-For"); forwardedFor != "" {
		ips := strings.Split(forwardedFor, ",")
		if len(ips) > 0 {
			ip := strings.TrimSpace(ips[0])
			logger.Debug("从X-Forwarded-For获取到IP: %s", ip)
			return ip
		}
	}

	// 3. 从RemoteAddr获取
	ip, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		// 如果分割失败,说明可能没有端口号,直接使用完整地址
		logger.Debug("从RemoteAddr获取到IP(无端口): %s", r.RemoteAddr)
		return r.RemoteAddr
	}

	logger.Debug("从RemoteAddr获取到IP: %s", ip)
	return ip
}

// HandleQueryIP 处理指定IP查询请求
func (h *IPHandler) HandleQueryIP(w http.ResponseWriter, r *http.Request) {
	ip := r.PathValue("ip")
	h.handleIPLookup(w, ip)
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
