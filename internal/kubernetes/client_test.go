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
)

func TestAllocationRequestNoRetry(t *testing.T) {
	var requests atomic.Int32

	server := httptest.NewServer(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			requests.Add(1)

			w.Header().Set("Retry-After", "1")
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusServiceUnavailable)

			fmt.Fprint(w, `{
				"apiVersion": "v1",
				"kind": "Status",
				"status": "Failure",
				"reason": "ServiceUnavailable",
				"message": "API unavailable",
				"code": 503
			}`)
		}),
	)
	defer server.Close()

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
`, server.URL)

	path := filepath.Join(t.TempDir(), "kubeconfig.yaml")

	if err := os.WriteFile(path, []byte(kubeconfig), 0600); err != nil {
		t.Fatal(err)
	}

	client, err := NewClient(path)
	if err != nil {
		t.Fatal(err)
	}

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

	ctx, cancel := context.WithTimeout(
		context.Background(),
		4*time.Second,
	)
	defer cancel()

	_, err = client.Resource(resource).
		Namespace("team-1").
		Create(ctx, allocation, metav1.CreateOptions{})

	if err == nil {
		t.Fatal("expected allocation error")
	}

	if count := requests.Load(); count != 1 {
		t.Fatalf(
			"expected exactly one HTTP request, got %d",
			count,
		)
	}
}
