#!/usr/bin/env bash
# 本 lab 的 Go 工具链放在 _tools/go-install 下（因为系统没装 Go）。
# 这里用 Windows 风格的绝对路径，MSYS Bash 里 Go 只认这种 GOROOT。
LABDIR="E:/项目/并行与分布式系统/MIT 6.5840/LEC1/lab"
export GOROOT="$LABDIR/_tools/go-install/go"
export PATH="$GOROOT/bin:$PATH"
export GOPATH="$LABDIR/_tools/gopath"
export GOCACHE="$LABDIR/_tools/gocache"
export GOTMPDIR="$LABDIR/_tools/gotmp"
mkdir -p "$GOCACHE" "$GOTMPDIR" "$GOPATH"
export GOFLAGS=-mod=mod
export GOPROXY=https://goproxy.cn,direct
cd "$LABDIR/src"
exec go.exe "$@"
