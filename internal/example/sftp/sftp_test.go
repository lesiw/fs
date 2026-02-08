package sftp

import (
	"fmt"
	"os"
	"runtime"
	"testing"
	"time"

	"golang.org/x/crypto/ssh"

	"lesiw.io/ctrctl"
	"lesiw.io/defers"
	"lesiw.io/fs"
	"lesiw.io/fs/fstest"
)

var testAddr string

func TestMain(m *testing.M) {
	if os.Getenv("CI") != "" {
		if runtime.GOOS == "windows" {
			fmt.Fprintln(os.Stderr, "skip: windows containers unsupported")
			return
		}
		if _, err := ctrctl.Version(nil); err != nil {
			fmt.Fprintln(os.Stderr, "skip: no container runtime available")
			return
		}
	}
	// Start SFTP server container
	addr, err := setupSFTP()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to setup SFTP: %v\n", err)
		defers.Exit(1)
	}
	testAddr = addr

	defers.Exit(m.Run())
}

func TestSFTPFS(t *testing.T) {
	if testAddr == "" {
		t.Skip("SFTP not available")
	}

	// Create SFTP filesystem
	fsys, err := New(testAddr, "testuser", "testpass")
	if err != nil {
		t.Fatalf("Failed to create SFTP filesystem: %v", err)
	}
	t.Cleanup(func() { _ = fs.Close(fsys) })

	// atmoz/sftp server chroots users to /home/testuser
	// The "upload" directory is relative to the chroot
	if sfs, ok := fsys.(*sftpFS); ok {
		sfs.SetBasePath("upload")
	}

	ctx := t.Context()

	// Run the fstest suite
	fstest.TestFS(ctx, t, fsys)
}

// setupSFTP starts an SFTP server container and returns the address.
// Cleanup is registered with defers.Add().
func setupSFTP() (string, error) {
	// atmoz/sftp uses user:pass:uid:gid:directories format
	// testuser:testpass:1001:1001:upload - creates user with upload directory
	id, err := ctrctl.ContainerCreate(&ctrctl.ContainerCreateOpts{
		Publish: []string{"22"},
	}, "atmoz/sftp:latest", "testuser:testpass:1001:1001:upload")
	if err != nil {
		return "", fmt.Errorf("create sftp container: %w", err)
	}
	defers.Add(func() {
		_, _ = ctrctl.ContainerRm(&ctrctl.ContainerRmOpts{Force: true}, id)
	})

	_, err = ctrctl.ContainerStart(nil, id)
	if err != nil {
		return "", fmt.Errorf("start sftp container: %w", err)
	}

	// Get mapped port
	var port string
	for range 50 {
		time.Sleep(100 * time.Millisecond)
		port, err = ctrctl.ContainerInspect(&ctrctl.ContainerInspectOpts{
			Format: `{{range $p, $conf := .NetworkSettings.Ports}}` +
				`{{if eq $p "22/tcp"}}` +
				`{{(index $conf 0).HostPort}}{{end}}{{end}}`,
		}, id)
		if err == nil && port != "" {
			break
		}
	}
	if port == "" {
		return "", fmt.Errorf("no port mapping found for 22/tcp")
	}

	addr := "localhost:" + port

	// Wait for SFTP to be ready - test connection
	config := &ssh.ClientConfig{
		User: "testuser",
		Auth: []ssh.AuthMethod{
			ssh.Password("testpass"),
		},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		Timeout:         5 * time.Second,
	}

	for range 50 {
		time.Sleep(200 * time.Millisecond)

		conn, err := ssh.Dial("tcp", addr, config)
		if err == nil {
			_ = conn.Close()
			return addr, nil
		}
	}

	return "", fmt.Errorf("sftp did not become ready in time")
}
