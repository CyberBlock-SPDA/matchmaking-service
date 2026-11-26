package kubernetes

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
)

func newTestClient(
	t *testing.T,
	serverURL string,
) dynamic.Interface {
	t.Helper()

	kubeconfig := fmt.Sprintf(`
apiVersion: v1
kind: Config
clusters:
- name: test
  cluster:
    server: %s
contexts:
- name: test
  context:
    cluster: test
    user: test
current-context: test
users:
- name: test
  user: {}
`, serverURL)

	path := filepath.Join(t.TempDir(), "kubeconfig.yaml")

	if err := os.WriteFile(path, []byte(kubeconfig), 0600); err != nil {
		t.Fatal(err)
	}

	client, err := NewClient(path)
	if err != nil {
		t.Fatal(err)
	}

	return client
}

func createTestAllocation(
	ctx context.Context,
	client dynamic.Interface,
) error {
	resource := schema.GroupVersionResource{
		Group:    "allocation.agones.dev",
		Version:  "v1",
		Resource: "gameserverallocations",
	}

	allocation := &unstructured.Unstructured{
		Object: map[string]any{
			"apiVersion": "allocation.agones.dev/v1",
			"kind":       "GameServerAllocation",
		},
	}

	_, err := client.Resource(resource).
		Namespace("team-1").
		Create(ctx, allocation, metav1.CreateOptions{})

	return err
}

func TestAllocationRequestNoRetry(t *testing.T) {
	tests := []struct {
		name       string
		statusCode int
		reason     string
		retryAfter string
	}{
		{
			name:       "service unavailable",
			statusCode: http.StatusServiceUnavailable,
			reason:     "ServiceUnavailable",
			retryAfter: "1",
		},
		{
			name:       "too many requests",
			statusCode: http.StatusTooManyRequests,
			reason:     "TooManyRequests",
			retryAfter: "1",
		},
		{
			name:       "internal server error",
			statusCode: http.StatusInternalServerError,
			reason:     "InternalError",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var requests atomic.Int32

			server := httptest.NewServer(
				http.HandlerFunc(func(
					w http.ResponseWriter,
					r *http.Request,
				) {
					requests.Add(1)

					if tt.retryAfter != "" {
						w.Header().Set(
							"Retry-After",
							tt.retryAfter,
						)
					}

					w.Header().Set(
						"Content-Type",
						"application/json",
					)

					w.WriteHeader(tt.statusCode)

					fmt.Fprintf(w, `{
						"apiVersion": "v1",
						"kind": "Status",
						"status": "Failure",
						"reason": %q,
						"message": "Test API error",
						"code": %d
					}`, tt.reason, tt.statusCode)
				}),
			)
			defer server.Close()

			client := newTestClient(t, server.URL)

			ctx, cancel := context.WithTimeout(
				context.Background(),
				4*time.Second,
			)
			defer cancel()

			err := createTestAllocation(ctx, client)

			if err == nil {
				t.Fatal("expected allocation error")
			}

			if count := requests.Load(); count != 1 {
				t.Fatalf(
					"expected exactly one HTTP request, got %d",
					count,
				)
			}
		})
	}
}

func TestAllocationTimeoutNoRetry(t *testing.T) {
	var requests atomic.Int32
	release := make(chan struct{})

	server := httptest.NewServer(
		http.HandlerFunc(func(
			w http.ResponseWriter,
			r *http.Request,
		) {
			requests.Add(1)

			select {
			case <-r.Context().Done():
			case <-release:
			}
		}),
	)
	defer server.Close()
	defer close(release)

	client := newTestClient(t, server.URL)

	ctx, cancel := context.WithTimeout(
		context.Background(),
		500*time.Millisecond,
	)
	defer cancel()

	err := createTestAllocation(ctx, client)

	if err == nil {
		t.Fatal("expected timeout error")
	}

	if ctx.Err() != context.DeadlineExceeded {
		t.Fatalf("expected deadline exceeded, got %v", ctx.Err())
	}

	if count := requests.Load(); count != 1 {
		t.Fatalf(
			"expected exactly one HTTP request, got %d",
			count,
		)
	}
}

func TestAllocationConnectionClosedNoRetry(t *testing.T) {
	var requests atomic.Int32

	server := httptest.NewServer(
		http.HandlerFunc(func(
			w http.ResponseWriter,
			r *http.Request,
		) {
			requests.Add(1)

			hijacker, ok := w.(http.Hijacker)
			if !ok {
				w.WriteHeader(http.StatusInternalServerError)
				return
			}

			conn, _, err := hijacker.Hijack()
			if err != nil {
				return
			}

			conn.Close()
		}),
	)
	defer server.Close()

	client := newTestClient(t, server.URL)

	ctx, cancel := context.WithTimeout(
		context.Background(),
		4*time.Second,
	)
	defer cancel()

	err := createTestAllocation(ctx, client)

	if err == nil {
		t.Fatal("expected connection error")
	}

	if count := requests.Load(); count != 1 {
		t.Fatalf(
			"expected exactly one HTTP request, got %d",
			count,
		)
	}
}
