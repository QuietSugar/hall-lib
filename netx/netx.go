// Package netx 提供面向中国网络的 GitHub 镜像 / 代理处理：
// 加速前缀解析（显式 / 自动探测）、URL 重写，以及带超时的 HTTP 取用。
//
// 语义与 ubuntu-server-init 的 utils.sh::detect_gh_proxy / gh_url 保持一致：
// 显式 GH_PROXY 优先；否则直连可用即直连；直连不通再依次试 GH_PROXY_AUTO_LIST。
package netx

import (
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"strings"
	"time"
)

// Canary 用于探测 GitHub 可达性（rupa/z 的 z.sh，很小且稳定）。
const Canary = "https://raw.githubusercontent.com/rupa/z/master/z.sh"

// DefaultAutoList 是默认候选加速站。
var DefaultAutoList = []string{"https://gh-proxy.com/", "https://ghfast.top/"}

var githubPrefixes = []string{
	"https://raw.githubusercontent.com/",
	"https://github.com/",
	"https://codeload.github.com/",
}

// Config 描述镜像策略。
type Config struct {
	Proxy     string        // 显式加速前缀（末尾斜杠可省）；非空则直接使用，不再探测
	Auto      bool          // Proxy 为空时是否自动探测
	AutoList  []string      // 候选加速站（Auto 为 true 时按序探测）
	Timeout   time.Duration // 单次探测 / 请求超时；<=0 用默认值
	CanaryURL string        // 可达性探测用的 URL；留空用 Canary（主要供测试注入）
}

// ConfigFromEnv 从 GH_PROXY / GH_PROXY_AUTO / GH_PROXY_AUTO_LIST 装载。
func ConfigFromEnv() Config {
	cfg := Config{Proxy: os.Getenv("GH_PROXY"), Auto: os.Getenv("GH_PROXY_AUTO") != "0"}
	if v := strings.TrimSpace(os.Getenv("GH_PROXY_AUTO_LIST")); v != "" {
		cfg.AutoList = strings.Fields(v)
	} else {
		cfg.AutoList = DefaultAutoList
	}
	return cfg
}

// GHURL 只重写 GitHub 域，其它 URL 原样返回。
func GHURL(raw, proxy string) string {
	if proxy == "" {
		return raw
	}
	for _, p := range githubPrefixes {
		if strings.HasPrefix(raw, p) {
			return strings.TrimRight(proxy, "/") + "/" + raw
		}
	}
	return raw
}

// ProbeResult 是一次探测的结果。
type ProbeResult struct {
	URL     string
	OK      bool
	Status  int
	Elapsed time.Duration
	Err     string
}

func timeoutOr(d, def time.Duration) time.Duration {
	if d <= 0 {
		return def
	}
	return d
}

func httpClient(timeout time.Duration) *http.Client {
	return &http.Client{
		Timeout: timeout,
		Transport: &http.Transport{
			Proxy:       http.ProxyFromEnvironment,
			DialContext: (&net.Dialer{Timeout: 3 * time.Second}).DialContext,
		},
	}
}

// Probe 请求 url，返回状态码与耗时。
func Probe(ctx context.Context, url string, timeout time.Duration) ProbeResult {
	timeout = timeoutOr(timeout, 6*time.Second)
	start := time.Now()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return ProbeResult{URL: url, Err: err.Error()}
	}
	resp, err := httpClient(timeout).Do(req)
	elapsed := time.Since(start)
	if err != nil {
		return ProbeResult{URL: url, Elapsed: elapsed, Err: err.Error()}
	}
	defer func() { _ = resp.Body.Close() }()
	_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, 1<<16))
	return ProbeResult{URL: url, OK: resp.StatusCode >= 200 && resp.StatusCode < 300, Status: resp.StatusCode, Elapsed: elapsed}
}

// Detect 解析加速前缀：显式 Proxy → 直连探测 → 依次试 AutoList；返回空串表示直连。
func Detect(ctx context.Context, cfg Config) string {
	if cfg.Proxy != "" {
		return strings.TrimRight(cfg.Proxy, "/")
	}
	if !cfg.Auto {
		return ""
	}
	canary := cfg.CanaryURL
	if canary == "" {
		canary = Canary
	}
	if Probe(ctx, canary, cfg.Timeout).OK {
		return ""
	}
	for _, cand := range cfg.AutoList {
		c := strings.TrimRight(cand, "/")
		if Probe(ctx, c+"/"+canary, cfg.Timeout).OK {
			return c
		}
	}
	return ""
}

// Get 取一个小文件（上限 1 MiB）的内容；proxy 非空时套用加速前缀。
func Get(ctx context.Context, url, proxy string, timeout time.Duration) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, GHURL(url, proxy), nil)
	if err != nil {
		return nil, err
	}
	resp, err := httpClient(timeoutOr(timeout, 30*time.Second)).Do(req)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP %d: %s", resp.StatusCode, url)
	}
	return io.ReadAll(io.LimitReader(resp.Body, 1<<20))
}

// Download 下载 url 到 dest（权限 0755）；proxy 非空时套用加速前缀。
func Download(ctx context.Context, url, proxy, dest string, timeout time.Duration) (int64, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, GHURL(url, proxy), nil)
	if err != nil {
		return 0, err
	}
	resp, err := httpClient(timeoutOr(timeout, 120*time.Second)).Do(req)
	if err != nil {
		return 0, err
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		return 0, fmt.Errorf("HTTP %d: %s", resp.StatusCode, url)
	}
	out, err := os.OpenFile(dest, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o755)
	if err != nil {
		return 0, err
	}
	n, err := io.Copy(out, resp.Body)
	cerr := out.Close()
	if err != nil {
		return n, err
	}
	return n, cerr
}
