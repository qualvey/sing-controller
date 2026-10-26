package main

import (
	"runtime/debug"
	"strings"
)

// version 由 GoReleaser 或构建脚本通过 ldflags 注入；未注入时默认为 dev。
var version = "dev"

// buildHash 可由构建命令显式注入。普通 go build 则从 Go 编译器写入的
// vcs.revision 中读取，确保不依赖某个特定的构建脚本。
var buildHash string

func init() {
	if buildHash == "" {
		buildHash = vcsRevision()
	}
	version = withBuildHash(version, buildHash)
}

func vcsRevision() string {
	info, ok := debug.ReadBuildInfo()
	if !ok {
		return ""
	}
	for _, setting := range info.Settings {
		if setting.Key == "vcs.revision" {
			return setting.Value
		}
	}
	return ""
}

func withBuildHash(base, hash string) string {
	if hash == "" {
		return base
	}
	if len(hash) > 7 {
		hash = hash[:7]
	}
	if strings.HasSuffix(base, "-"+hash) {
		return base
	}
	return base + "-" + hash
}
