package handler

import (
	"encoding/json"
	"log"
	"net/http"

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
	// 获取真实IP
	headers := make(map[string]string)
	headers["X-Real-IP"] = r.Header.Get("X-Real-IP")
	headers["X-Forwarded-For"] = r.Header.Get("X-Forwarded-For")
	ip := service.GetRealIP(headers, r.RemoteAddr)

	h.handleIPLookup(w, ip)
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
			log.Printf("查询IP信息失败: %v", err)
			http.Error(w, "服务器内部错误", http.StatusInternalServerError)
		}
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		log.Printf("编码响应失败: %v", err)
		http.Error(w, "服务器内部错误", http.StatusInternalServerError)
		return
	}
}
