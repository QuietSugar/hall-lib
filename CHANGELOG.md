# Changelog

本项目遵循 [Keep a Changelog](https://keepachangelog.com/zh-CN/1.1.0/)，
版本号遵循 [语义化版本](https://semver.org/lang/zh-CN/)。

每次发布时：把 `[Unreleased]` 整理为对应版本号章节（含日期），并在其顶部保留新的空 `[Unreleased]` 章节。

## [Unreleased]

## [0.0.1] - 2026-10-18

### Added

- `netx` 共享网络库：`GH_PROXY` 探测分层（显式 → 直连探测 → `GH_PROXY_AUTO_LIST` 加速站 → 直连），配置走 `ConfigFromEnv`，结果缓存可测（`netx_test.go`）
- `GHURL` / `Detect` / `Probe` / `Get` / `Download`：给各消费方统一 GitHub URL 重写与下载（含代理环境变量与超时控制）
