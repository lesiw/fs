package smb

import (
	"fmt"
	"os"
	"testing"
	"time"

	"lesiw.io/ctrctl"
	"lesiw.io/defers"
	"lesiw.io/fs/fstest"
)

var testAddr string

func TestMain(m *testing.M) {
	// Start Samba server container
	addr, err := setupSMB()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to setup SMB: %v\n", err)
		defers.Exit(1)
	}
	testAddr = addr

	defers.Exit(m.Run())
}

func TestSMBFS(t *testing.T) {
	if testAddr == "" {
		t.Skip("SMB not available")
	}

	// Create SMB filesystem
	fsys, err := New(testAddr, "public", "testuser", "testpass")
	if err != nil {
		t.Fatalf("Failed to create SMB filesystem: %v", err)
	}
	t.Cleanup(func() { _ = fsys.Close() })

	ctx := t.Context()

	// Run the fstest suite
	fstest.TestFS(ctx, t, fsys)
}

// setupSMB starts a Samba server container and returns the address.
// Cleanup is registered with defers.Add().
func setupSMB() (string, error) {
	// Create container with user and share configuration
	// -p: Set ownership/permissions
	// -u "user;password": Create user
	// -s "share;/mount;browsable;readonly;guest;users": writable share
	//    Format: name;path;browsable;readonly;guest;users
	//    browsable=yes, readonly=no (writable), guest=no
	id, err := ctrctl.ContainerCreate(&ctrctl.ContainerCreateOpts{
		Publish: []string{"445"},
	}, "dperson/samba:latest", "-p", "-u", "testuser;testpass",
		"-s", "public;/mount;yes;no;no;testuser")
	if err != nil {
		return "", fmt.Errorf("create smb container: %w", err)
	}
	defers.Add(func() {
		_, _ = ctrctl.ContainerRm(&ctrctl.ContainerRmOpts{Force: true}, id)
	})

	_, err = ctrctl.ContainerStart(nil, id)
	if err != nil {
		return "", fmt.Errorf("start smb container: %w", err)
	}

	// Get mapped port
	var port string
	for range 50 {
		time.Sleep(100 * time.Millisecond)
		port, err = ctrctl.ContainerInspect(&ctrctl.ContainerInspectOpts{
			Format: `{{range $p, $conf := .NetworkSettings.Ports}}` +
				`{{if eq $p "445/tcp"}}` +
				`{{(index $conf 0).HostPort}}{{end}}{{end}}`,
		}, id)
		if err == nil && port != "" {
			break
		}
	}
	if port == "" {
		return "", fmt.Errorf("no port mapping found for 445/tcp")
	}

	addr := "localhost:" + port

	// Wait for Samba to be ready - test connection
	for range 50 {
		time.Sleep(200 * time.Millisecond)

		_, err := New(addr, "public", "testuser", "testpass")
		if err == nil {
			return addr, nil
		}
	}

	return "", fmt.Errorf("smb did not become ready in time")
}
