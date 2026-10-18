package netx

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestGHURL(t *testing.T) {
	cases := []struct{ in, proxy, want string }{
		{"https://raw.githubusercontent.com/a/b", "", "https://raw.githubusercontent.com/a/b"},
		{"https://raw.githubusercontent.com/a/b", "https://gh-proxy.com/", "https://gh-proxy.com/https://raw.githubusercontent.com/a/b"},
		{"https://github.com/a/b", "https://gh-proxy.com", "https://gh-proxy.com/https://github.com/a/b"},
		{"https://codeload.github.com/x", "https://gh-proxy.com/", "https://gh-proxy.com/https://codeload.github.com/x"},
		{"https://mirrors.aliyun.com/x", "https://gh-proxy.com/", "https://mirrors.aliyun.com/x"},
		{"http://example.com/repository/x", "https://gh-proxy.com/", "http://example.com/repository/x"},
	}
	for _, c := range cases {
		if got := GHURL(c.in, c.proxy); got != c.want {
			t.Errorf("GHURL(%q, %q) = %q, want %q", c.in, c.proxy, got, c.want)
		}
	}
}

func TestProbe(t *testing.T) {
	ok := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(200) }))
	defer ok.Close()
	if r := Probe(context.Background(), ok.URL, 3*time.Second); !r.OK || r.Status != 200 {
		t.Fatalf("Probe(200) = %+v", r)
	}
	bad := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(500) }))
	defer bad.Close()
	if r := Probe(context.Background(), bad.URL, 3*time.Second); r.OK {
		t.Fatalf("Probe(500) 应失败: %+v", r)
	}
}

func TestDetect(t *testing.T) {
	direct := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(200) }))
	defer direct.Close()
	if got := Detect(context.Background(), Config{Auto: true, CanaryURL: direct.URL}); got != "" {
		t.Fatalf("直连可用时应返回空串，得到 %q", got)
	}
	dead := httptest.NewServer(http.HandlerFunc(func(_ http.ResponseWriter, _ *http.Request) {}))
	deadURL := dead.URL
	dead.Close()
	good := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(200) }))
	defer good.Close()
	cfg := Config{Auto: true, AutoList: []string{deadURL + "/", good.URL + "/"}, CanaryURL: deadURL + "/canary", Timeout: 2 * time.Second}
	if got := Detect(context.Background(), cfg); got != good.URL {
		t.Fatalf("应回退到可用加速站 %q，得到 %q", good.URL, got)
	}
	if got := Detect(context.Background(), Config{Proxy: "https://gh-proxy.com/"}); got != "https://gh-proxy.com" {
		t.Fatalf("显式 Proxy 应生效，得到 %q", got)
	}
	if got := Detect(context.Background(), Config{}); got != "" {
		t.Fatalf("Auto=false 应直连，得到 %q", got)
	}
}

func TestGetDownload(t *testing.T) {
	payload := []byte("hello-hall-lib")
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/small" {
			_, _ = w.Write(payload)
			return
		}
		http.NotFound(w, r)
	}))
	defer srv.Close()
	ctx := context.Background()
	got, err := Get(ctx, srv.URL+"/small", "", 5*time.Second)
	if err != nil || string(got) != string(payload) {
		t.Fatalf("Get = %q, %v", string(got), err)
	}
	if _, err := Get(ctx, srv.URL+"/missing", "", 5*time.Second); err == nil {
		t.Fatal("404 应报错")
	}
	dest := filepath.Join(t.TempDir(), "out")
	n, err := Download(ctx, srv.URL+"/small", "", dest, 5*time.Second)
	if err != nil || n != int64(len(payload)) {
		t.Fatalf("Download = %d, %v", n, err)
	}
	b, _ := os.ReadFile(dest)
	if string(b) != string(payload) {
		t.Fatalf("文件内容 = %q", string(b))
	}
}
