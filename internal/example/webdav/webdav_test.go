package webdav

import (
	"fmt"
	"os"
	"testing"
	"time"

	"lesiw.io/ctrctl"
	"lesiw.io/defers"
	"lesiw.io/fs/fstest"
)

var testURL string

func TestMain(m *testing.M) {
	// Start WebDAV container
	url, err := setupWebDAV()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to setup WebDAV: %v\n", err)
		defers.Exit(1)
	}
	testURL = url

	defers.Exit(m.Run())
}

func TestWebDAVFS(t *testing.T) {
	if testURL == "" {
		t.Skip("WebDAV not available")
	}

	// Create WebDAV filesystem
	fsys, err := New(testURL, "testuser", "testpass")
	if err != nil {
		t.Fatalf("Failed to create WebDAV filesystem: %v", err)
	}

	ctx := t.Context()

	// Run the fstest suite
	fstest.TestFS(ctx, t, fsys)
}

// setupWebDAV starts a WebDAV container and returns the URL.
// Cleanup is registered with defers.Add().
func setupWebDAV() (string, error) {
	// Create temporary config file
	configContent := `address: 0.0.0.0
port: 8080
directory: /data
users:
  - username: testuser
    password: testpass
    permissions: CRUD
    rules: []
`
	tmpfile, err := os.CreateTemp("", "webdav-config-*.yml")
	if err != nil {
		return "", fmt.Errorf("create temp config file: %w", err)
	}
	tmpfilePath := tmpfile.Name()
	defers.Add(func() {
		_ = os.Remove(tmpfilePath)
	})

	if _, err := tmpfile.Write([]byte(configContent)); err != nil {
		_ = tmpfile.Close()
		return "", fmt.Errorf("write config file: %w", err)
	}
	_ = tmpfile.Close()

	// Create Docker volume for data
	volumeName := "webdav-test-" + fmt.Sprintf("%d", time.Now().UnixNano())
	_, err = ctrctl.VolumeCreate(nil, volumeName)
	if err != nil {
		return "", fmt.Errorf("create volume: %w", err)
	}
	defers.Add(func() {
		_, _ = ctrctl.VolumeRm(&ctrctl.VolumeRmOpts{Force: true}, volumeName)
	})

	// Create container with volume mounts for config and data
	id, err := ctrctl.ContainerCreate(&ctrctl.ContainerCreateOpts{
		Publish: []string{"8080"},
		Volume: []string{
			tmpfilePath + ":/config.yml:ro",
			volumeName + ":/data",
		},
	}, "hacdias/webdav:latest", "-c", "/config.yml")
	if err != nil {
		return "", fmt.Errorf("create webdav container: %w", err)
	}
	defers.Add(func() {
		_, _ = ctrctl.ContainerRm(&ctrctl.ContainerRmOpts{Force: true}, id)
	})

	_, err = ctrctl.ContainerStart(nil, id)
	if err != nil {
		return "", fmt.Errorf("start webdav container: %w", err)
	}

	// Wait for network ports to be available
	var port string
	for range 50 {
		time.Sleep(100 * time.Millisecond)
		port, err = ctrctl.ContainerInspect(&ctrctl.ContainerInspectOpts{
			Format: `{{range $p, $conf := .NetworkSettings.Ports}}` +
				`{{if eq $p "8080/tcp"}}` +
				`{{(index $conf 0).HostPort}}{{end}}{{end}}`,
		}, id)
		if err == nil && port != "" {
			break
		}
	}
	if port == "" {
		return "", fmt.Errorf(
			"no port mapping found for 8080/tcp after 50 attempts",
		)
	}

	url := "http://localhost:" + port

	// Wait for WebDAV to be ready - test connection
	for range 50 {
		time.Sleep(200 * time.Millisecond)

		_, err := New(url, "testuser", "testpass")
		if err == nil {
			return url, nil
		}
	}

	return "", fmt.Errorf("webdav did not become ready in time")
}
