package kubernetes

import (
	"fmt"
	"net/http"
	"strings"

	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

func NewClient(kubeconfigPath string) (dynamic.Interface, error) {
	var config *rest.Config
	var err error

	if kubeconfigPath != "" {
		config, err = clientcmd.BuildConfigFromFlags(
			"",
			kubeconfigPath,
		)
	} else {
		config, err = rest.InClusterConfig()
	}

	if err != nil {
		return nil, fmt.Errorf(
			"load Kubernetes config: %w", err,
		)
	}

	config.Wrap(func(rt http.RoundTripper) http.RoundTripper {
		return &allocationTransport{
			next: rt,
		}
	})

	client, err := dynamic.NewForConfig(config)
	if err != nil {
		return nil, fmt.Errorf(
			"create Kubernetes client: %w", err,
		)
	}

	return client, nil
}

type allocationTransport struct {
	next http.RoundTripper
}

func (t *allocationTransport) RoundTrip(
	req *http.Request,
) (*http.Response, error) {
	resp, err := t.next.RoundTrip(req)

	if err != nil {
		return resp, err
	}

	if req.Method == http.MethodPost &&
		strings.HasPrefix(
			req.URL.Path,
			"/apis/allocation.agones.dev/v1/namespaces/",
		) &&
		strings.HasSuffix(
			req.URL.Path,
			"/gameserverallocations",
		) {
		resp.Header.Del("Retry-After")
	}

	return resp, nil
}
