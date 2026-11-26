package allocator

import (
	"context"
	"errors"
	"testing"
	"time"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/dynamic/fake"
	ktesting "k8s.io/client-go/testing"

	"github.com/CyberBlock-SPDA/matchmaking-service/internal/matchmaking"
)

func allocatedResponse() *unstructured.Unstructured {
	return &unstructured.Unstructured{
		Object: map[string]any{
			"apiVersion": "allocation.agones.dev/v1",
			"kind":       "GameServerAllocation",
			"status": map[string]any{
				"state":          "Allocated",
				"gameServerName": "doom-fleet-test",
				"address":        "172.18.0.2",
				"ports": []any{
					map[string]any{
						"name": "webrtc",
						"port": int64(7325),
					},
					map[string]any{
						"name": "signaling",
						"port": int64(7061),
					},
				},
			},
		},
	}
}

func TestAgonesAllocatorSuccess(t *testing.T) {
	client := fake.NewSimpleDynamicClient(
		runtime.NewScheme(),
	)

	client.PrependReactor(
		"create",
		"gameserverallocations",
		func(action ktesting.Action) (
			bool, runtime.Object, error,
		) {
			return true, allocatedResponse(), nil
		},
	)

	a := NewAgonesAllocator(
		client,
		"team-1",
		"doom-fleet",
		5*time.Second,
	)

	server, err := a.Allocate(
		context.Background(),
		"doom-1993",
	)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if server.Host != "172.18.0.2" {
		t.Errorf("unexpected host: %s", server.Host)
	}

	if server.Signaling.Port != 7061 {
		t.Errorf("unexpected signaling port: %d",
			server.Signaling.Port)
	}

	if server.Signaling.Protocol != "wss" {
		t.Error("unexpected signaling protocol")
	}

	if server.Signaling.Path != "/ws" {
		t.Error("unexpected signaling path")
	}

	if server.WebRTC.Port != 7325 {
		t.Errorf("unexpected WebRTC port: %d",
			server.WebRTC.Port)
	}

	if server.WebRTC.Protocol != "udp" {
		t.Error("unexpected WebRTC protocol")
	}

	actions := client.Actions()

	if len(actions) != 1 {
		t.Fatalf("expected one API request, got %d",
			len(actions))
	}

	action := actions[0]

	if action.GetNamespace() != "team-1" {
		t.Error("unexpected namespace")
	}

	if action.GetResource() != allocationGVR {
		t.Error("unexpected API resource")
	}

	createAction, ok := action.(ktesting.CreateAction)
	if !ok {
		t.Fatal("expected create action")
	}

	object, ok := createAction.GetObject().(*unstructured.Unstructured)
	if !ok {
		t.Fatal("expected unstructured object")
	}

	selectors, found, err := unstructured.NestedSlice(
		object.Object, "spec", "selectors",
	)

	if err != nil || !found || len(selectors) != 1 {
		t.Fatal("invalid selectors")
	}

	selector, ok := selectors[0].(map[string]any)
	if !ok {
		t.Fatal("invalid selector")
	}

	fleet, _, _ := unstructured.NestedString(
		selector, "matchLabels", "agones.dev/fleet",
	)

	if fleet != "doom-fleet" {
		t.Errorf("unexpected fleet: %s", fleet)
	}

	state, _, _ := unstructured.NestedString(
		selector, "gameServerState",
	)

	if state != "Ready" {
		t.Errorf("unexpected state: %s", state)
	}

	scheduling, _, _ := unstructured.NestedString(
		object.Object, "spec", "scheduling",
	)

	if scheduling != "Packed" {
		t.Errorf("unexpected scheduling: %s", scheduling)
	}
}

func TestAgonesAllocatorNoServers(t *testing.T) {
	for _, state := range []string{"UnAllocated", "Contention"} {
		t.Run(state, func(t *testing.T) {
			client := fake.NewSimpleDynamicClient(
				runtime.NewScheme(),
			)

			response := &unstructured.Unstructured{
				Object: map[string]any{
					"status": map[string]any{
						"state": state,
					},
				},
			}

			client.PrependReactor(
				"create",
				"gameserverallocations",
				func(action ktesting.Action) (
					bool, runtime.Object, error,
				) {
					return true, response, nil
				},
			)

			a := NewAgonesAllocator(
				client,
				"team-1",
				"doom-fleet",
				5*time.Second,
			)

			_, err := a.Allocate(
				context.Background(),
				"doom-1993",
			)

			if !errors.Is(err, matchmaking.ErrNoAvailableServers) {
				t.Fatalf("expected ErrNoAvailableServers, got %v", err)
			}
		})
	}
}

func TestAgonesAllocatorAPIError(t *testing.T) {
	client := fake.NewSimpleDynamicClient(
		runtime.NewScheme(),
	)

	client.PrependReactor(
		"create",
		"gameserverallocations",
		func(action ktesting.Action) (
			bool, runtime.Object, error,
		) {
			return true, nil, errors.New("API unavailable")
		},
	)

	a := NewAgonesAllocator(
		client,
		"team-1",
		"doom-fleet",
		5*time.Second,
	)

	_, err := a.Allocate(
		context.Background(),
		"doom-1993",
	)

	if !errors.Is(err, matchmaking.ErrAllocationFailed) {
		t.Fatalf("expected ErrAllocationFailed, got %v", err)
	}

	if len(client.Actions()) != 1 {
		t.Fatal("allocation must not be retried by allocator")
	}
}

func TestAgonesAllocatorMissingAddress(t *testing.T) {
	response := allocatedResponse()

	unstructured.RemoveNestedField(
		response.Object, "status", "address",
	)

	_, err := parseAllocation(response)

	if !errors.Is(err, matchmaking.ErrAllocationFailed) {
		t.Fatalf("expected ErrAllocationFailed, got %v", err)
	}
}

func TestAgonesAllocatorMissingPort(t *testing.T) {
	response := allocatedResponse()

	err := unstructured.SetNestedSlice(
		response.Object,
		[]any{
			map[string]any{
				"name": "signaling",
				"port": int64(7061),
			},
		},
		"status", "ports",
	)

	if err != nil {
		t.Fatal(err)
	}

	_, err = parseAllocation(response)

	if !errors.Is(err, matchmaking.ErrAllocationFailed) {
		t.Fatalf("expected ErrAllocationFailed, got %v", err)
	}
}

func TestAgonesAllocatorInvalidPort(t *testing.T) {
	response := allocatedResponse()

	err := unstructured.SetNestedSlice(
		response.Object,
		[]any{
			map[string]any{
				"name": "signaling",
				"port": int64(70000),
			},
			map[string]any{
				"name": "webrtc",
				"port": int64(7325),
			},
		},
		"status", "ports",
	)

	if err != nil {
		t.Fatal(err)
	}

	_, err = parseAllocation(response)

	if !errors.Is(err, matchmaking.ErrAllocationFailed) {
		t.Fatalf("expected ErrAllocationFailed, got %v", err)
	}
}

func TestAgonesAllocatorUnsupportedGame(t *testing.T) {
	client := fake.NewSimpleDynamicClient(
		runtime.NewScheme(),
	)

	a := NewAgonesAllocator(
		client,
		"team-1",
		"doom-fleet",
		5*time.Second,
	)

	_, err := a.Allocate(
		context.Background(),
		"unknown-game",
	)

	if !errors.Is(err, matchmaking.ErrAllocationFailed) {
		t.Fatalf("expected ErrAllocationFailed, got %v", err)
	}

	if len(client.Actions()) != 0 {
		t.Fatal("unsupported game must not trigger allocation")
	}
}

func TestAgonesAllocatorCanceled(t *testing.T) {
	client := fake.NewSimpleDynamicClient(
		runtime.NewScheme(),
	)

	a := NewAgonesAllocator(
		client,
		"team-1",
		"doom-fleet",
		5*time.Second,
	)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := a.Allocate(ctx, "doom-1993")

	if !errors.Is(err, context.Canceled) {
		t.Fatalf("expected context.Canceled, got %v", err)
	}

	if len(client.Actions()) != 0 {
		t.Fatal("canceled request must not trigger allocation")
	}
}
