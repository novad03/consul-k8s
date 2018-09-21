package catalog

import (
	"context"
	"fmt"
	"time"

	"github.com/cenkalti/backoff"
	"github.com/hashicorp/consul/api"
	"github.com/hashicorp/go-hclog"
)

// Source is the source for the sync that watches Consul services and
// updates a Sink whenever the set of services to register changes.
type Source struct {
	Client *api.Client  // Consul API client
	Domain string       // Consul DNS domain
	Sink   Sink         // Sink is the sink to update with services
	Log    hclog.Logger // Logger
}

// Run is the long-running runloop for watching Consul services and
// updating the Sink.
func (s *Source) Run(ctx context.Context) {
	opts := (&api.QueryOptions{
		AllowStale: true,
		WaitIndex:  1,
		WaitTime:   1 * time.Minute,
	}).WithContext(ctx)
	for {
		// Get all services with tags.
		var serviceMap map[string][]string
		var meta *api.QueryMeta
		err := backoff.Retry(func() error {
			var err error
			serviceMap, meta, err = s.Client.Catalog().Services(opts)
			return err
		}, backoff.WithContext(backoff.NewExponentialBackOff(), ctx))

		// If the context is ended, then we end
		if ctx.Err() != nil {
			return
		}

		// If there was an error, handle that
		if err != nil {
			s.Log.Warn("error querying services, will retry", "err", err)
			continue
		}

		// Update our blocking index
		opts.WaitIndex = meta.LastIndex

		// Setup the services
		services := make(map[string]string, len(serviceMap))
		for name, _ := range serviceMap {
			services[name] = fmt.Sprintf("%s.service.%s.", name, s.Domain)
		}
		s.Log.Info("received services from Consul", "count", len(services))

		// Lock so we can modify the
		s.Sink.SetServices(services)
	}
}
