# hall-lib

QuietSugar 的个人通用 Go 依赖库（按领域分包，纯标准库优先）。

## 包

| 包 | 作用 |
| --- | --- |
| [`netx`](netx/) | 面向中国网络的 GitHub 镜像 / 代理：加速前缀解析（显式 / 自动探测）、URL 重写、带超时的取用 |

## 用法

```go
cfg := netx.ConfigFromEnv()
proxy := netx.Detect(ctx, cfg)          // "" 表示直连
url := netx.GHURL(rawGitHubURL, proxy)  // 只重写 GitHub 域
data, err := netx.Get(ctx, url, proxy, 0)
```

## 约定

- 纯标准库优先；新增第三方依赖需说明理由。
- `make check` = gofmt + go vet + go test。
