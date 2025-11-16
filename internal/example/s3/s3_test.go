package s3

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"

	"lesiw.io/ctrctl"
	"lesiw.io/defers"
	"lesiw.io/fs/fstest"
)

var testEndpoint string

func TestMain(m *testing.M) {
	// Start MinIO container
	endpoint, err := setupMinIO()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to setup MinIO: %v\n", err)
		defers.Exit(1)
	}
	testEndpoint = endpoint

	defers.Exit(m.Run())
}

func TestS3FS(t *testing.T) {
	if testEndpoint == "" {
		t.Skip("MinIO not available")
	}

	// Create S3 filesystem
	fsys, err := New(
		testEndpoint, "test-bucket", "minioadmin", "minioadmin", false,
	)
	if err != nil {
		t.Fatalf("Failed to create S3 filesystem: %v", err)
	}

	ctx := t.Context()

	// Run the fstest suite
	fstest.TestFS(ctx, t, fsys)
}

// setupMinIO starts a MinIO container and returns the endpoint.
// Cleanup is registered with defers.Add().
func setupMinIO() (string, error) {
	id, err := ctrctl.ContainerCreate(&ctrctl.ContainerCreateOpts{
		Env: []string{
			"MINIO_ROOT_USER=minioadmin",
			"MINIO_ROOT_PASSWORD=minioadmin",
		},
		Publish: []string{"9000", "9001"},
	},
		"quay.io/minio/minio:latest",
		"server", "/data", "--console-address", ":9001",
	)
	if err != nil {
		return "", fmt.Errorf("create minio container: %w", err)
	}
	defers.Add(func() {
		_, _ = ctrctl.ContainerRm(&ctrctl.ContainerRmOpts{Force: true}, id)
	})

	_, err = ctrctl.ContainerStart(nil, id)
	if err != nil {
		return "", fmt.Errorf("start minio container: %w", err)
	}

	// Wait for network ports to be available
	var port string
	for range 50 {
		time.Sleep(100 * time.Millisecond)
		port, err = ctrctl.ContainerInspect(&ctrctl.ContainerInspectOpts{
			Format: `{{range $p, $conf := .NetworkSettings.Ports}}` +
				`{{if eq $p "9000/tcp"}}` +
				`{{(index $conf 0).HostPort}}{{end}}{{end}}`,
		}, id)
		if err == nil && port != "" {
			break
		}
	}
	if port == "" {
		return "", fmt.Errorf(
			"no port mapping found for 9000/tcp after 50 attempts",
		)
	}

	endpoint := "localhost:" + port

	// Wait for MinIO to be ready and create bucket
	for range 50 {
		time.Sleep(200 * time.Millisecond)

		client, err := minio.New(endpoint, &minio.Options{
			Creds:  credentials.NewStaticV4("minioadmin", "minioadmin", ""),
			Secure: false,
		})
		if err != nil {
			continue
		}

		ctx, cancel := context.WithTimeout(
			context.Background(), 2*time.Second,
		)
		_, err = client.ListBuckets(ctx)
		cancel()
		if err != nil {
			continue
		}

		// MinIO is ready, create test bucket
		ctx, cancel = context.WithTimeout(
			context.Background(), 2*time.Second,
		)
		err = client.MakeBucket(ctx, "test-bucket", minio.MakeBucketOptions{})
		cancel()
		if err != nil {
			// Bucket might already exist, check if we can access it
			ctx, cancel = context.WithTimeout(
				context.Background(), 2*time.Second,
			)
			exists, checkErr := client.BucketExists(ctx, "test-bucket")
			cancel()
			if checkErr != nil || !exists {
				return "", fmt.Errorf("create test bucket: %w", err)
			}
		}
		return endpoint, nil
	}

	return "", fmt.Errorf("minio did not become ready in time")
}
