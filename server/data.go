package server

import (
	"embed"
	"net/http"

	"github.com/sonnt85/vfs"
)

//go:embed html/**
var embedfs embed.FS
var sembedfs *vfs.VFS
var inited bool

func Init() {
	if inited {
		return
	}
	defer func() {
		inited = true
	}()
	sembedfs, _ = vfs.NewEmbedHttpSystemFS(&embedfs, "html")
	StaticHandler = http.FileServer(sembedfs)
}
