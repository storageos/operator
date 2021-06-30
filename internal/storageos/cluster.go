package storageos

import (
	"context"
	"reflect"

	api "github.com/storageos/go-api/v2"
)

// Cluster is the configuration of a StorageOS cluster.
type Cluster struct {
	DisableTelemetry      bool
	DisableCrashReporting bool
	DisableVersionCheck   bool
	LogLevel              string
	LogFormat             string
	Version               string
}

// IsEqual checks if a cluster is equal to a given cluster in interms of
// configuration.
func (c *Cluster) IsEqual(b *Cluster) bool {
	return reflect.DeepEqual(c, b)
}

// GetCluster fetches StorageOS configuration and returns a Cluster with the
// fetched data.
func (c *Client) GetCluster(ctx context.Context) (*Cluster, error) {
	ctx = c.AddToken(ctx)

	cluster, resp, err := c.api.GetCluster(ctx)
	if err != nil {
		return nil, api.MapAPIError(err, resp)
	}
	return &Cluster{
		DisableTelemetry:      cluster.DisableTelemetry,
		DisableCrashReporting: cluster.DisableCrashReporting,
		DisableVersionCheck:   cluster.DisableVersionCheck,
		LogLevel:              string(cluster.LogLevel),
		LogFormat:             string(cluster.LogFormat),
		Version:               cluster.Version,
	}, nil
}

// UpdateCluster updates the configuration of a cluster.
func (c *Client) UpdateCluster(ctx context.Context, cluster *Cluster) error {
	ctx = c.AddToken(ctx)

	data := api.UpdateClusterData{
		DisableTelemetry:      cluster.DisableTelemetry,
		DisableCrashReporting: cluster.DisableCrashReporting,
		DisableVersionCheck:   cluster.DisableVersionCheck,
		LogLevel:              api.LogLevel(cluster.LogLevel),
		LogFormat:             api.LogFormat(cluster.LogFormat),
		Version:               cluster.Version,
	}
	_, resp, err := c.api.UpdateCluster(ctx, data, &api.UpdateClusterOpts{})
	if err != nil {
		return api.MapAPIError(err, resp)
	}
	return nil
}
