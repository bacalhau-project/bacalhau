package analytics

import (
	"go.opentelemetry.io/otel/attribute"
)

const (
	InstallationIDKey = "installation_id"
	InstanceIDKey     = "instance_id"
	NodeIDKey         = "node_id"
	NodeTypeKey       = "node_type"
)

type Config struct {
	attributes   []attribute.KeyValue
	otlpEndpoint string
}

// Option is a functional option for configuring the LogRecorder instance
type Option func(*Config) error

func WithNodeNodeID(id string) Option {
	return func(c *Config) error {
		c.attributes = append(c.attributes, attribute.String(NodeIDKey, id))
		return nil
	}
}

func WithNodeType(isRequester, isCompute bool) Option {
	return func(c *Config) error {
		var typ string
		if isRequester && isCompute {
			typ = "hybrid"
		} else if isRequester {
			typ = "orchestrator"
		} else if isCompute {
			typ = "compute"
		}
		c.attributes = append(c.attributes, attribute.String(NodeTypeKey, typ))
		return nil
	}
}

func WithInstallationID(id string) Option {
	return func(c *Config) error {
		c.attributes = append(c.attributes, attribute.String(InstallationIDKey, id))
		return nil
	}
}

func WithInstanceID(id string) Option {
	return func(c *Config) error {
		c.attributes = append(c.attributes, attribute.String(InstanceIDKey, id))
		return nil
	}
}
