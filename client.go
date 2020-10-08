package reactor

import (
	"net/http"

	"k8s.io/client-go/kubernetes"
)

// Client holds http and kubernetes clients
type Client struct {
	HTTP       http.Client
	Kubernetes kubernetes.Clientset
}
