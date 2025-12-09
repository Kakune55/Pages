package adminui

import (
	"embed"
	"io/fs"
)

var (
    //go:embed admin
    files embed.FS
)

// FS returns the embedded filesystem containing the admin UI assets.
func FS() fs.FS {
    return files
}
