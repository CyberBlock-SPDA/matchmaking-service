package allocator

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/CyberBlock-SPDA/matchmaking-service/internal/matchmaking"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic/fake"
	ktesting "k8s.io/client-go/testing"
)

func testGameServer(name, sessionID, state string) *unstructured.Unstructured {
	server := &unstructured.Unstructured{
		Object: map[string]any{
			"apiVersion": "agones.dev/v1",
			"kind":       "GameServer",
			"metadata": map[string]any{
				"name":      name,
				"namespace": "team-1",
			},
			"status": map[string]any{"state": state},
		},
	}
	server.SetLabels(map[string]string{"agones.dev/fleet": "doom-fleet"})
	server.SetAnnotations(map[string]string{sessionAnnotation: sessionID})
	return server
}

func terminationClient(t *testing.T, objects ...runtime.Object) *fake.FakeDynamicClient {
	t.Helper()

	return fake.NewSimpleDynamicClientWithCustomListKinds(
		runtime.NewScheme(),
		map[schema.GroupVersionResource]string{
			gameServerGVR: "GameServerList",
		},
		objects...,
	)
}

func TestTerminateSessionDeletesAllocatedGameServer(t *testing.T) {
	client := terminationClient(
		t,
		testGameServer("doom-fleet-test", "session-test", "Allocated"),
		testGameServer("doom-fleet-other", "other-session", "Allocated"),
	)
	agones := NewAgonesAllocator(client, "team-1", "doom-fleet", time.Second)

	if err := agones.TerminateSession(context.Background(), "session-test"); err != nil {
		t.Fatalf("terminate session: %v", err)
	}

	var deleted []string
	for _, action := range client.Actions() {
		if action.GetVerb() == "delete" {
			deleteAction, ok := action.(ktesting.DeleteAction)
			if !ok {
				t.Fatalf("unexpected delete action type %T", action)
			}
			deleted = append(deleted, deleteAction.GetName())
		}
	}
	if len(deleted) != 1 || deleted[0] != "doom-fleet-test" {
		t.Fatalf("unexpected deleted GameServers: %v", deleted)
	}
}

func TestTerminateSessionNotFound(t *testing.T) {
	agones := NewAgonesAllocator(terminationClient(t), "team-1", "doom-fleet", time.Second)

	err := agones.TerminateSession(context.Background(), "missing")
	if !errors.Is(err, matchmaking.ErrSessionNotFound) {
		t.Fatalf("expected session not found, got %v", err)
	}
}

func TestTerminateSessionRejectsNonAllocatedGameServer(t *testing.T) {
	client := terminationClient(
		t,
		testGameServer("doom-fleet-test", "session-test", "Ready"),
	)
	agones := NewAgonesAllocator(client, "team-1", "doom-fleet", time.Second)

	err := agones.TerminateSession(context.Background(), "session-test")
	if !errors.Is(err, matchmaking.ErrSessionNotActive) {
		t.Fatalf("expected session not active, got %v", err)
	}

	for _, action := range client.Actions() {
		if action.GetVerb() == "delete" {
			t.Fatal("did not expect an unallocated GameServer to be deleted")
		}
	}
}

func TestTerminateSessionDoesNotMatchAnotherFleet(t *testing.T) {
	server := testGameServer("other-fleet-server", "session-test", "Allocated")
	server.SetLabels(map[string]string{"agones.dev/fleet": "other-fleet"})
	agones := NewAgonesAllocator(terminationClient(t, server), "team-1", "doom-fleet", time.Second)

	err := agones.TerminateSession(context.Background(), "session-test")
	if !errors.Is(err, matchmaking.ErrSessionNotFound) {
		t.Fatalf("expected session not found, got %v", err)
	}
}
