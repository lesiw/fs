package ssh

import (
	"fmt"
	"os"
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
	// Start SSH server container
	addr, err := setupSSH()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to setup SSH: %v\n", err)
		defers.Exit(1)
	}
	testAddr = addr

	defers.Exit(m.Run())
}

func TestSSHFS(t *testing.T) {
	if testAddr == "" {
		t.Skip("SSH not available")
	}

	// Create SSH filesystem
	fsys, err := New(testAddr, "testuser", "testpass")
	if err != nil {
		t.Fatalf("Failed to create SSH filesystem: %v", err)
	}
	t.Cleanup(func() { _ = fs.Close(fsys) })

	// linuxserver/openssh-server puts files in /config which is the
	// user's home directory. Use a subdirectory for testing.
	if sfs, ok := fsys.(*sshFS); ok {
		sfs.SetPrefix("testdir")
	}

	ctx := t.Context()

	// Run the fstest suite
	fstest.TestFS(ctx, t, fsys)
}

// setupSSH starts an SSH server container and returns the address.
// Cleanup is registered with defers.Add().
func setupSSH() (string, error) {
	id, err := ctrctl.ContainerCreate(&ctrctl.ContainerCreateOpts{
		Env: []string{
			"USER_NAME=testuser",
			"USER_PASSWORD=testpass",
			"PASSWORD_ACCESS=true",
		},
		Publish: []string{"2222"},
	}, "linuxserver/openssh-server:latest", "")
	if err != nil {
		return "", fmt.Errorf("create ssh container: %w", err)
	}
	defers.Add(func() {
		_, _ = ctrctl.ContainerRm(&ctrctl.ContainerRmOpts{Force: true}, id)
	})

	_, err = ctrctl.ContainerStart(nil, id)
	if err != nil {
		return "", fmt.Errorf("start ssh container: %w", err)
	}

	port, err := ctrctl.ContainerInspect(&ctrctl.ContainerInspectOpts{
		Format: `{{range $p, $conf := .NetworkSettings.Ports}}` +
			`{{if eq $p "2222/tcp"}}` +
			`{{(index $conf 0).HostPort}}{{end}}{{end}}`,
	}, id)
	if err != nil {
		return "", fmt.Errorf("get ssh port: %w", err)
	}
	if port == "" {
		return "", fmt.Errorf("no port mapping found for 2222/tcp")
	}

	addr := "localhost:" + port

	// Wait for SSH to be ready
	config := &ssh.ClientConfig{
		User: "testuser",
		Auth: []ssh.AuthMethod{
			ssh.Password("testpass"),
		},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
	}

	for i := range 50 {
		if i > 0 {
			// Sleep between retry attempts (SSH takes longer to start)
			time.Sleep(500 * time.Millisecond)
		}

		conn, err := ssh.Dial("tcp", addr, config)
		if err == nil {
			_ = conn.Close()
			return addr, nil
		}
	}

	return "", fmt.Errorf("ssh did not become ready in time")
}
