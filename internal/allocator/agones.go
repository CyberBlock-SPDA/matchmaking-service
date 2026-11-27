package allocator

import (
	"context"
	"fmt"
	"strings"
	"time"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"

	"github.com/CyberBlock-SPDA/matchmaking-service/internal/matchmaking"
	"github.com/CyberBlock-SPDA/matchmaking-service/internal/models"
)

const sessionAnnotation = "cloudgaming.spda/session-id"

var allocationGVR = schema.GroupVersionResource{
	Group:    "allocation.agones.dev",
	Version:  "v1",
	Resource: "gameserverallocations",
}

var gameServerGVR = schema.GroupVersionResource{
	Group:    "agones.dev",
	Version:  "v1",
	Resource: "gameservers",
}

type AgonesAllocator struct {
	client    dynamic.Interface
	namespace string
	fleetName string
	timeout   time.Duration
}

func NewAgonesAllocator(
	client dynamic.Interface,
	namespace string,
	fleetName string,
	timeout time.Duration,
) *AgonesAllocator {
	return &AgonesAllocator{
		client:    client,
		namespace: namespace,
		fleetName: fleetName,
		timeout:   timeout,
	}
}

func (a *AgonesAllocator) Allocate(
	ctx context.Context,
	gameID string,
	sessionID string,
) (models.GameServer, error) {
	if gameID != "doom-1993" {
		return models.GameServer{}, fmt.Errorf(
			"%w: unsupported game %q",
			matchmaking.ErrAllocationFailed,
			gameID,
		)
	}

	if strings.TrimSpace(sessionID) == "" {
		return models.GameServer{}, fmt.Errorf(
			"%w: session ID is empty",
			matchmaking.ErrAllocationFailed,
		)
	}

	ctx, cancel := context.WithTimeout(ctx, a.timeout)
	defer cancel()

	if err := ctx.Err(); err != nil {
		return models.GameServer{}, fmt.Errorf(
			"%w: %w",
			matchmaking.ErrAllocationFailed,
			err,
		)
	}

	allocation := a.buildAllocation(sessionID)

	result, err := a.client.
		Resource(allocationGVR).
		Namespace(a.namespace).
		Create(ctx, allocation, metav1.CreateOptions{})

	if err != nil {
		return models.GameServer{}, fmt.Errorf(
			"%w: Kubernetes API request failed: %w",
			matchmaking.ErrAllocationFailed,
			err,
		)
	}

	return parseAllocation(result)
}

func (a *AgonesAllocator) TerminateSession(
	ctx context.Context,
	sessionID string,
) error {
	if strings.TrimSpace(sessionID) == "" {
		return matchmaking.ErrSessionNotFound
	}

	ctx, cancel := context.WithTimeout(ctx, a.timeout)
	defer cancel()

	servers, err := a.client.
		Resource(gameServerGVR).
		Namespace(a.namespace).
		List(ctx, metav1.ListOptions{})
	if err != nil {
		return fmt.Errorf("list Fleet GameServers: %w", err)
	}

	var matching *unstructured.Unstructured
	for i := range servers.Items {
		server := &servers.Items[i]
		if server.GetLabels()["agones.dev/fleet"] != a.fleetName {
			continue
		}
		if server.GetAnnotations()[sessionAnnotation] != sessionID {
			continue
		}
		if matching != nil {
			return fmt.Errorf(
				"multiple GameServers found for session %q",
				sessionID,
			)
		}
		matching = server
	}

	if matching == nil {
		return matchmaking.ErrSessionNotFound
	}

	state, found, err := unstructured.NestedString(
		matching.Object, "status", "state",
	)
	if err != nil {
		return fmt.Errorf("read GameServer state: %w", err)
	}
	if !found || state != "Allocated" {
		return matchmaking.ErrSessionNotActive
	}

	err = a.client.
		Resource(gameServerGVR).
		Namespace(a.namespace).
		Delete(ctx, matching.GetName(), metav1.DeleteOptions{})
	if apierrors.IsNotFound(err) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("delete allocated GameServer: %w", err)
	}

	return nil
}

func (a *AgonesAllocator) buildAllocation(
	sessionID string,
) *unstructured.Unstructured {
	return &unstructured.Unstructured{
		Object: map[string]any{
			"apiVersion": "allocation.agones.dev/v1",
			"kind":       "GameServerAllocation",
			"metadata": map[string]any{
				"generateName": "doom-allocation-",
				"namespace":    a.namespace,
			},
			"spec": map[string]any{
				"selectors": []any{
					map[string]any{
						"matchLabels": map[string]any{
							"agones.dev/fleet": a.fleetName,
						},
						"gameServerState": "Ready",
					},
				},
				"scheduling": "Packed",
				"metadata": map[string]any{
					"annotations": map[string]any{
						sessionAnnotation: sessionID,
					},
				},
			},
		},
	}
}

func parseAllocation(
	result *unstructured.Unstructured,
) (models.GameServer, error) {
	if result == nil {
		return models.GameServer{}, fmt.Errorf(
			"%w: empty allocation response",
			matchmaking.ErrAllocationFailed,
		)
	}

	state, found, err := unstructured.NestedString(
		result.Object, "status", "state",
	)

	if err != nil || !found {
		return models.GameServer{}, fmt.Errorf(
			"%w: missing allocation state",
			matchmaking.ErrAllocationFailed,
		)
	}

	switch state {
	case "UnAllocated", "Contention":
		return models.GameServer{},
			matchmaking.ErrNoAvailableServers

	case "Allocated":
		// Continue processing.

	default:
		return models.GameServer{}, fmt.Errorf(
			"%w: unexpected allocation state %q",
			matchmaking.ErrAllocationFailed,
			state,
		)
	}

	host, found, err := unstructured.NestedString(
		result.Object, "status", "address",
	)

	if err != nil || !found || strings.TrimSpace(host) == "" {
		return models.GameServer{}, fmt.Errorf(
			"%w: missing server address",
			matchmaking.ErrAllocationFailed,
		)
	}

	ports, found, err := unstructured.NestedSlice(
		result.Object, "status", "ports",
	)

	if err != nil || !found {
		return models.GameServer{}, fmt.Errorf(
			"%w: missing server ports",
			matchmaking.ErrAllocationFailed,
		)
	}

	signalingPort, err := findPort(ports, "signaling")
	if err != nil {
		return models.GameServer{}, err
	}

	webrtcPort, err := findPort(ports, "webrtc")
	if err != nil {
		return models.GameServer{}, err
	}

	return models.GameServer{
		Host: host,
		Signaling: models.SignalingEndpoint{
			Protocol: "wss",
			Port:     signalingPort,
			Path:     "/ws",
		},
		WebRTC: models.WebRTCEndpoint{
			Protocol: "udp",
			Port:     webrtcPort,
		},
	}, nil
}

func findPort(ports []any, name string) (int, error) {
	for _, item := range ports {
		port, ok := item.(map[string]any)
		if !ok {
			continue
		}

		portName, _, _ := unstructured.NestedString(
			port, "name",
		)

		if portName != name {
			continue
		}

		number, found, err := unstructured.NestedInt64(
			port, "port",
		)

		if err != nil || !found ||
			number < 1 || number > 65535 {
			return 0, fmt.Errorf(
				"%w: invalid port %q",
				matchmaking.ErrAllocationFailed,
				name,
			)
		}

		return int(number), nil
	}

	return 0, fmt.Errorf(
		"%w: required port %q not found",
		matchmaking.ErrAllocationFailed,
		name,
	)
}

var _ matchmaking.Allocator = (*AgonesAllocator)(nil)
var _ matchmaking.SessionTerminator = (*AgonesAllocator)(nil)
