package downloader

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"sync"
	"time"

	"ip-geo/internal/logger"
)

var mmdbFiles = map[string]string{
	"mmdb/GeoIP2-City.mmdb":   "https://pan.dnslin.com/d/pan/GeoIP2-City.mmdb",
	"mmdb/GeoLite2-ASN.mmdb":  "https://github.com/P3TERX/GeoLite.mmdb/raw/download/GeoLite2-ASN.mmdb",
	"mmdb/GeoCN.mmdb":         "http://github.com/ljxi/GeoCN/releases/download/Latest/GeoCN.mmdb",
}

// EnsureMMDBFiles 确保所有必需的MMDB文件存在，如果不存在则下载
func EnsureMMDBFiles() error {
	// 创建mmdb目录
	if err := os.MkdirAll("mmdb", 0755); err != nil {
		return fmt.Errorf("创建mmdb目录失败: %v", err)
	}

	var wg sync.WaitGroup
	errChan := make(chan error, len(mmdbFiles))

	for filePath, url := range mmdbFiles {
		if !fileExists(filePath) {
			wg.Add(1)
			go func(fp, u string) {
				defer wg.Done()
				logger.Info("开始下载数据库: %s", fp)
				if err := downloadFileWithRetry(u, fp); err != nil {
					errChan <- fmt.Errorf("下载文件 %s 失败: %v", fp, err)
					return
				}
				logger.Info("数据库下载完成: %s", fp)
			}(filePath, url)
		} else {
			logger.Debug("数据库文件已存在: %s", filePath)
		}
	}

	// 等待所有下载完成
	go func() {
		wg.Wait()
		close(errChan)
	}()

	// 收集错误
	for err := range errChan {
		if err != nil {
			return err
		}
	}

	return nil
}

// fileExists 检查文件是否存在
func fileExists(filename string) bool {
	info, err := os.Stat(filename)
	if os.IsNotExist(err) {
		return false
	}
	return !info.IsDir()
}

// downloadFileWithRetry 带重试的文件下载
func downloadFileWithRetry(url, filepath string) error {
	maxRetries := 3
	var lastErr error

	for i := 0; i < maxRetries; i++ {
		if i > 0 {
			logger.Info("重试下载 %s (第 %d 次)", filepath, i+1)
			time.Sleep(time.Second * time.Duration(i+1)) // 递增延迟
		}

		if err := downloadFileWithProgress(url, filepath); err != nil {
			lastErr = err
			logger.Warn("下载失败 %s: %v", filepath, err)
			continue
		}
		return nil
	}

	return fmt.Errorf("达到最大重试次数: %v", lastErr)
}

// downloadFileWithProgress 带进度的文件下载
func downloadFileWithProgress(url, filepath string) error {
	// 创建临时文件
	tmpFile := filepath + ".tmp"
	out, err := os.Create(tmpFile)
	if err != nil {
		return err
	}
	defer out.Close()

	// 发送GET请求
	client := &http.Client{
		Timeout: 30 * time.Second,
	}
	resp, err := client.Get(url)
	if err != nil {
		os.Remove(tmpFile)
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		os.Remove(tmpFile)
		return fmt.Errorf("下载失败，状态码: %d", resp.StatusCode)
	}

	// 获取文件大小
	fileSize := resp.ContentLength

	// 创建进度读取器
	reader := &ProgressReader{
		Reader:     resp.Body,
		Total:      fileSize,
		FilePath:   filepath,
		LastUpdate: time.Now(),
	}

	// 复制响应内容到文件
	_, err = io.Copy(out, reader)
	if err != nil {
		os.Remove(tmpFile)
		return err
	}

	// 关闭临时文件
	out.Close()

	// 重命名临时文件为目标文件
	if err := os.Rename(tmpFile, filepath); err != nil {
		os.Remove(tmpFile)
		return err
	}

	return nil
}

// ProgressReader 进度读取器
type ProgressReader struct {
	Reader     io.Reader
	Total      int64
	Current    int64
	FilePath   string
	LastUpdate time.Time
}

// Read 实现io.Reader接口
func (pr *ProgressReader) Read(p []byte) (int, error) {
	n, err := pr.Reader.Read(p)
	pr.Current += int64(n)

	// 每秒更新一次进度
	if time.Since(pr.LastUpdate) > time.Second {
		if pr.Total > 0 {
			progress := float64(pr.Current) / float64(pr.Total) * 100
			logger.Info("下载进度 %s: %.2f%% (%d/%d bytes)", 
				pr.FilePath, progress, pr.Current, pr.Total)
		} else {
			logger.Info("下载进度 %s: %d bytes", pr.FilePath, pr.Current)
		}
		pr.LastUpdate = time.Now()
	}

	return n, err
} 