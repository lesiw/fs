module lesiw.io/fs/internal/example/webdav

go 1.24.2

require (
	github.com/Antonboom/errname v1.1.1
	github.com/studio-b12/gowebdav v0.11.0
	golang.org/x/tools v0.39.0
	lesiw.io/checker v0.12.0
	lesiw.io/ctrctl v0.14.0
	lesiw.io/defers v0.9.0
	lesiw.io/errcheck v1.0.0
	lesiw.io/fs v0.0.0
	lesiw.io/linelen v0.2.0
	lesiw.io/plscheck v0.20.0
	lesiw.io/tidytypes v0.2.0
)

require (
	golang.org/x/mod v0.30.0 // indirect
	golang.org/x/sync v0.18.0 // indirect
	golang.org/x/sys v0.38.0 // indirect
	golang.org/x/telemetry v0.0.0-20251111182119-bc8e575c7b54 // indirect
)

replace lesiw.io/fs => ../../..
