module lesiw.io/fs/internal/example/webdav

go 1.24.2

require (
	github.com/studio-b12/gowebdav v0.11.0
	lesiw.io/ctrctl v0.14.0
	lesiw.io/defers v0.9.0
	lesiw.io/fs v0.0.0
)

require (
	github.com/google/go-cmp v0.7.0 // indirect
	golang.org/x/net v0.47.0 // indirect
)

replace lesiw.io/fs => ../../..
