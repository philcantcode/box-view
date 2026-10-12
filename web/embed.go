// Package web embeds a dependency-free HTML adapter and demo application.
package web

import (
	"embed"
	"io/fs"
	"net/http"
)

//go:embed index.html box-view.js app.js style.css
var files embed.FS

func Handler() http.Handler { sub, _ := fs.Sub(files, "."); return http.FileServer(http.FS(sub)) }
