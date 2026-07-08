// Package web embeds the server-rendered templates and static assets
// (CSS/JS/images) shipped inside the server binary.
package web

import "embed"

//go:embed templates
var Templates embed.FS

// Static holds only the build-time assets (CSS/JS). Uploaded media is
// runtime-writable and lives on disk instead — see internal/config
// UploadsDir — because embed.FS is compiled in and can't grow at runtime.
//
//go:embed static/css static/js static/img
var Static embed.FS
