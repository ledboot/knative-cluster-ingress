package v1

import (
	"github.com/hbagdi/go-kong/kong"
	extensions "k8s.io/api/extensions/v1beta1"
)

// Route represents a Kong Route and holds a reference to the Ingress
// rule.
type Route struct {
	kong.Route
	// Ingress object associated with this route
	Ingress extensions.Ingress
	Plugins []kong.Plugin
}

// Service represents a service in Kong and holds routes associated with the
// service and other k8s metadata.
type Service struct {
	kong.Service
	Backend   extensions.IngressBackend
	Namespace string
	Routes    []Route
	Plugins   []kong.Plugin
}

// Upstream is a wrapper around Upstream object in Kong.
type Upstream struct {
	kong.Upstream
	Targets []Target
	// Service this upstream is asosciated with.
	Service Service
}

// Target is a wrapper around Target object in Kong.
type Target struct {
	kong.Target
}

// Consumer holds a Kong consumer and it's plugins and credentials.
type Consumer struct {
	kong.Consumer
	Plugins []kong.Plugin
	// Credentials type(key-auth, basic-auth) to credential mapping
	Credentials map[string][]map[string]interface{}
}

// KongState holds the configuration that should be applied to Kong.
type KongState struct {
	Services      []Service
	Upstreams     []Upstream
	Certificates  []Certificate
	GlobalPlugins []Plugin
	Consumers     []Consumer
}

type Certificate struct {
	kong.Certificate
}

type Plugin struct {
	kong.Plugin
}


