module lesiw.io/fs/internal/example/sftp

go 1.24.2

require (
	github.com/pkg/sftp v1.13.10
	golang.org/x/crypto v0.44.0
	lesiw.io/ctrctl v0.14.0
	lesiw.io/defers v0.9.0
	lesiw.io/fs v0.0.0
)

require (
	github.com/google/go-cmp v0.7.0 // indirect
	github.com/kr/fs v0.1.0 // indirect
	golang.org/x/sys v0.38.0 // indirect
)

replace lesiw.io/fs => ../../../
