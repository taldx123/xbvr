package ui

import (
	"embed"
	"io/fs"
	"log"
	"net/http"
)

//go:embed dist
var Assets embed.FS

func GetFileSystem() http.FileSystem {

	fs, err := fs.Sub(Assets, "dist")
	if err != nil {
		log.Panic(err)
	}
	return http.FS(fs)
}
