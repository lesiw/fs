module lesiw.io/fs/internal/example/sftp

go 1.24.7

require (
	github.com/Antonboom/errname v1.1.1
	github.com/pkg/sftp v1.13.10
	golang.org/x/crypto v0.44.0
	golang.org/x/tools v0.38.0
	lesiw.io/checker v0.11.0
	lesiw.io/ctrctl v0.14.0
	lesiw.io/defers v0.9.0
	lesiw.io/errcheck v1.0.0
	lesiw.io/fs v0.0.0
	lesiw.io/linelen v0.1.0
	lesiw.io/plscheck v0.20.0
	lesiw.io/tidytypes v0.1.0
)

require (
	github.com/kr/fs v0.1.0 // indirect
	golang.org/x/mod v0.29.0 // indirect
	golang.org/x/sync v0.17.0 // indirect
	golang.org/x/sys v0.38.0 // indirect
	golang.org/x/telemetry v0.0.0-20251008203120-078029d740a8 // indirect
)

replace lesiw.io/fs => ../../../
