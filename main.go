package main

import (
	"errors"
	"flag"
	"io"
	"log"
	"net/http"
	"os"
	"path"
	"strconv"
	"strings"
	"sync"
)

var (
	port               int  // 代理端口
	enableProxyAnySite bool // 是否打开代理任意站点的路由
)

// 缓冲池
var bufferPool = sync.Pool{
	New: func() interface{} {
		return make([]byte, 1024) // 定义缓冲区的大小
	},
}

func main() {
	// 从命令行参数获取端口配置
	flag.IntVar(&port, "port", 9000, "The port on which the service runs.")
	flag.BoolVar(&enableProxyAnySite, "enable-proxy-any-site", false, "Enable any site proxy router.")
	flag.Parse()

	// 打印配置信息
	log.Printf("INFO: Service running on http://127.0.0.1:%d (Press CTRL+C to quit)", port)

	// 设置路由处理器
	http.HandleFunc("/", handleRequest)

	// 启动服务器
	log.Fatal(http.ListenAndServe(":"+strconv.Itoa(port), nil))
}

// handleRequest 处理所有请求，根据地址匹配来选择执行的函数
func handleRequest(w http.ResponseWriter, r *http.Request) {
	switch {
	case r.URL.Path == "/":
		// 处理根路径
		content, err := os.ReadFile("index.html")
		if err != nil {
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			log.Printf("ERROR: reading index.html: %s", err.Error())
			return
		}
		w.Header().Set("Content-Type", "text/html")
		w.Write(content)
	case strings.HasPrefix(r.URL.Path, "/o"):
		proxyRequest(w, r, "https://api.openai.com")
	case strings.HasPrefix(r.URL.Path, "/c"):
		proxyRequest(w, r, "https://api.cloudflare.com")
	case strings.HasPrefix(r.URL.Path, "/p"):
		if enableProxyAnySite {
			target := r.Header.Get("X-Target-Host")
			if target == "" {
				http.Error(w, "Bad Request", http.StatusBadRequest)
				return
			}
			proxyRequest(w, r, "https://"+target)
		} else {
			http.Error(w, "The any site proxy function is disabled.", http.StatusForbidden)
		}
	default:
		http.NotFound(w, r)
	}
	return
}

// proxyRequest 处理代理请求的共用函数
func proxyRequest(w http.ResponseWriter, r *http.Request, target string) {
	// 重写URL，移除代理指示部分（如 /c, /o 或 /p）
	rewrittenPath, err := rewriteURLPath(r.URL.Path)
	if err != nil {
		log.Printf("ERROR: Invalid path [%s] while rewrite: %s", r.URL.Path, err.Error())
		http.Error(w, "Invalid request path", http.StatusBadRequest)
		return
	}

	// 拼接目标URL
	targetURL := target + rewrittenPath
	if r.URL.RawQuery != "" {
		targetURL += "?" + r.URL.RawQuery
	}
	// 打印代理请求信息，包含用户 IP
	logRequestInfo(r, targetURL)

	// 创建代理HTTP请求
	proxyReq, err := http.NewRequest(r.Method, targetURL, r.Body)
	if err != nil {
		log.Println("ERROR: creating proxy request: ", err.Error())
		http.Error(w, "Error creating proxy request", http.StatusInternalServerError)
		return
	}

	// 删除原始请求中的敏感头部
	r.Header.Del("X-Forwarded-For")
	r.Header.Del("X-Real-IP")

	// 将原始请求头复制到新请求中
	copyHeaders(r.Header, proxyReq.Header)

	// 发起代理请求
	sendProxyRequest(w, proxyReq)
}

// copyHeaders 复制请求头
func copyHeaders(src, dest http.Header) {
	for headerKey, headerValues := range src {
		for _, headerValue := range headerValues {
			dest.Add(headerKey, headerValue)
		}
	}
}

// sendProxyRequest 发送代理请求并处理响应
func sendProxyRequest(w http.ResponseWriter, req *http.Request) {
	client := &http.Client{}

	resp, err := client.Do(req)
	if err != nil {
		log.Println("ERROR: sending proxy request: ", err.Error())
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer resp.Body.Close()

	// 将响应头复制到代理响应头中
	copyHeaders(resp.Header, w.Header())

	// 设置响应状态码
	w.WriteHeader(resp.StatusCode)

	// 从池中获取缓冲区
	buf := bufferPool.Get().([]byte)
	defer bufferPool.Put(buf) // 操作完成后放回池中

	// 将响应实体写入到响应流中（支持流式响应）
	for {
		n, err := resp.Body.Read(buf)
		if err == io.EOF || n == 0 {
			return
		}
		if err != nil {
			log.Println("ERROR: reading response body: ", err.Error())
			http.Error(w, "Error reading response", http.StatusInternalServerError)
			return
		}
		if _, err = w.Write(buf[:n]); err != nil {
			log.Println("ERROR: writing response: ", err.Error())
			http.Error(w, "Error writing response", http.StatusInternalServerError)
			return
		}
		w.(http.Flusher).Flush()
	}
}

func logRequestInfo(r *http.Request, targetURL string) {
	// 提取 X-Forwarded-For 和 X-Real-IP 头部
	xForwardedFor := r.Header.Get("X-Forwarded-For")
	xRealIP := r.Header.Get("X-Real-IP")

	// 确定使用者IP
	userIP := ""
	if xForwardedFor != "" {
		userIP = xForwardedFor
	} else if xRealIP != "" {
		userIP = xRealIP
	} else {
		userIP = r.RemoteAddr
	}

	// 打印用户 IP 和最终代理的 URL
	log.Printf("INFO: Proxying request for [%s] to [%s]\n", userIP, targetURL)
}

func rewriteURLPath(rawPath string) (string, error) {
	// 使用 path 或 filepath 标准库进行路径清理
	cleanedPath := path.Clean(rawPath)

	// 分割路径
	pathParts := strings.SplitN(cleanedPath, "/", 3)

	// 检查分割后的路径部分数量
	// NOTE: 此处的逻辑会导致无法匹配根目录访问，也就是“/”的访问，但因为没什么影响，所以暂时不管啦
	if len(pathParts) < 3 {
		return "", errors.New("invalid path format")
	}

	// 构建新的路径
	newPath := "/" + pathParts[2]

	return newPath, nil
}
