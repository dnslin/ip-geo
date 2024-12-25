package middleware

import "net/http"

func CORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// 允许所有来源
		w.Header().Set("Access-Control-Allow-Origin", "*")
		// 允许的请求方法
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		// 允许的请求头
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		// 允许凭证
		w.Header().Set("Access-Control-Allow-Credentials", "true")

		// 处理 OPTIONS 预检请求
		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}
