package synccatalog

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"sync"
	"time"

	catalogFromConsul "github.com/hashicorp/consul-k8s/catalog/from-consul"
	catalogFromK8S "github.com/hashicorp/consul-k8s/catalog/from-k8s"
	"github.com/hashicorp/consul-k8s/helper/controller"
	"github.com/hashicorp/consul-k8s/subcommand"
	k8sflags "github.com/hashicorp/consul-k8s/subcommand/flags"
	"github.com/hashicorp/consul/command/flags"
	"github.com/hashicorp/go-hclog"
	"github.com/mitchellh/cli"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
)

// Command is the command for syncing the K8S and Consul service
// catalogs (one or both directions).
type Command struct {
	UI cli.Ui

	flags                     *flag.FlagSet
	http                      *flags.HTTPFlags
	k8s                       *k8sflags.K8SFlags
	flagToConsul              bool
	flagToK8S                 bool
	flagConsulDomain          string
	flagConsulK8STag          string
	flagK8SDefault            bool
	flagK8SServicePrefix      string
	flagK8SSourceNamespace    string
	flagK8SWriteNamespace     string
	flagConsulWritePeriod     flags.DurationValue
	flagSyncClusterIPServices bool
	flagNodePortSyncType      string

	once sync.Once
	help string
}

func (c *Command) init() {
	c.flags = flag.NewFlagSet("", flag.ContinueOnError)
	c.flags.BoolVar(&c.flagToConsul, "to-consul", true,
		"If true, K8S services will be synced to Consul.")
	c.flags.BoolVar(&c.flagToK8S, "to-k8s", true,
		"If true, Consul services will be synced to Kubernetes.")
	c.flags.BoolVar(&c.flagK8SDefault, "k8s-default-sync", true,
		"If true, all valid services in K8S are synced by default. If false, "+
			"the service must be annotated properly to sync. In either case "+
			"an annotation can override the default")
	c.flags.StringVar(&c.flagK8SServicePrefix, "k8s-service-prefix", "",
		"A prefix to prepend to all services written to Kubernetes from Consul. "+
			"If this is not set then services will have no prefix.")
	c.flags.StringVar(&c.flagK8SSourceNamespace, "k8s-source-namespace", metav1.NamespaceAll,
		"The Kubernetes namespace to watch for service changes and sync to Consul. "+
			"If this is not set then it will default to all namespaces.")
	c.flags.StringVar(&c.flagK8SWriteNamespace, "k8s-write-namespace", metav1.NamespaceDefault,
		"The Kubernetes namespace to write to for services from Consul. "+
			"If this is not set then it will default to the default namespace.")
	c.flags.StringVar(&c.flagConsulDomain, "consul-domain", "consul",
		"The domain for Consul services to use when writing services to "+
			"Kubernetes. Defaults to consul.")
	c.flags.StringVar(&c.flagConsulK8STag, "consul-k8s-tag", "k8s",
		"Tag value for K8S services registered in Consul")
	c.flags.Var(&c.flagConsulWritePeriod, "consul-write-interval",
		"The interval to perform syncing operations creating Consul services. "+
			"All changes are merged and write calls are only made on this "+
			"interval. Defaults to 30 seconds.")
	c.flags.BoolVar(&c.flagSyncClusterIPServices, "sync-clusterip-services", true,
		"If true, all valid ClusterIP services in K8S are synced by default. If false, "+
			"ClusterIP services are not synced to Consul.")
	c.flags.StringVar(&c.flagNodePortSyncType, "node-port-sync-type", "ExternalOnly",
		"Defines the type of sync for NodePort services. Valid options are ExternalOnly, "+
			"InternalOnly and ExternalFirst.")

	c.http = &flags.HTTPFlags{}
	c.k8s = &k8sflags.K8SFlags{}
	flags.Merge(c.flags, c.http.ClientFlags())
	flags.Merge(c.flags, c.http.ServerFlags())
	flags.Merge(c.flags, c.k8s.Flags())
	c.help = flags.Usage(help, c.flags)
}

func (c *Command) Run(args []string) int {
	c.once.Do(c.init)
	if err := c.flags.Parse(args); err != nil {
		return 1
	}
	if len(c.flags.Args()) > 0 {
		c.UI.Error(fmt.Sprintf("Should have no non-flag arguments."))
		return 1
	}

	config, err := subcommand.K8SConfig(c.k8s.KubeConfig())
	if err != nil {
		c.UI.Error(fmt.Sprintf("Error retrieving Kubernetes auth: %s", err))
		return 1
	}

	// create the clientset
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		c.UI.Error(fmt.Sprintf("Error initializing Kubernetes client: %s", err))
		return 1
	}

	// Setup Consul client
	consulClient, err := c.http.APIClient()
	if err != nil {
		c.UI.Error(fmt.Sprintf("Error connecting to Consul agent: %s", err))
		return 1
	}

	// Get the sync interval
	var syncInterval time.Duration
	c.flagConsulWritePeriod.Merge(&syncInterval)

	// Create the context we'll use to cancel everything
	ctx, cancelF := context.WithCancel(context.Background())

	// Start the K8S-to-Consul syncer
	var toConsulCh chan struct{}
	if c.flagToConsul {
		// Build the Consul sync and start it
		syncer := &catalogFromK8S.ConsulSyncer{
			Client:            consulClient,
			Log:               hclog.Default().Named("to-consul/sink"),
			Namespace:         c.flagK8SSourceNamespace,
			SyncPeriod:        syncInterval,
			ServicePollPeriod: syncInterval * 2,
			ConsulK8STag:      c.flagConsulK8STag,
		}
		go syncer.Run(ctx)

		// Build the controller and start it
		ctl := &controller.Controller{
			Log: hclog.Default().Named("to-consul/controller"),
			Resource: &catalogFromK8S.ServiceResource{
				Log:            hclog.Default().Named("to-consul/source"),
				Client:         clientset,
				Syncer:         syncer,
				Namespace:      c.flagK8SSourceNamespace,
				ExplicitEnable: !c.flagK8SDefault,
				ClusterIPSync:  c.flagSyncClusterIPServices,
				NodePortSync:   catalogFromK8S.NodePortSyncType(c.flagNodePortSyncType),
				ConsulK8STag:   c.flagConsulK8STag,
			},
		}

		toConsulCh = make(chan struct{})
		go func() {
			defer close(toConsulCh)
			ctl.Run(ctx.Done())
		}()
	}

	// Start Consul-to-K8S sync
	var toK8SCh chan struct{}
	if c.flagToK8S {
		sink := &catalogFromConsul.K8SSink{
			Client:    clientset,
			Namespace: c.flagK8SWriteNamespace,
			Log:       hclog.Default().Named("to-k8s/sink"),
		}

		source := &catalogFromConsul.Source{
			Client:       consulClient,
			Domain:       c.flagConsulDomain,
			Sink:         sink,
			Prefix:       c.flagK8SServicePrefix,
			Log:          hclog.Default().Named("to-k8s/source"),
			ConsulK8STag: c.flagConsulK8STag,
		}
		go source.Run(ctx)

		// Build the controller and start it
		ctl := &controller.Controller{
			Log:      hclog.Default().Named("to-k8s/controller"),
			Resource: sink,
		}

		toK8SCh = make(chan struct{})
		go func() {
			defer close(toK8SCh)
			ctl.Run(ctx.Done())
		}()
	}

	// Wait on an interrupt to exit
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt)
	select {
	// Unexpected exit
	case <-toConsulCh:
		cancelF()
		if toK8SCh != nil {
			<-toK8SCh
		}
		return 1

	// Unexpected exit
	case <-toK8SCh:
		cancelF()
		if toConsulCh != nil {
			<-toConsulCh
		}
		return 1

	// Interrupted, gracefully exit
	case <-sigCh:
		cancelF()
		if toConsulCh != nil {
			<-toConsulCh
		}
		if toK8SCh != nil {
			<-toK8SCh
		}
		return 0
	}
}

func (c *Command) Synopsis() string { return synopsis }
func (c *Command) Help() string {
	c.once.Do(c.init)
	return c.help
}

const synopsis = "Sync Kubernetes services and Consul services."
const help = `
Usage: consul-k8s sync-catalog [options]

  Sync K8S pods, services, and more with the Consul service catalog.
  This enables K8S services to discover and communicate with external
  services, and allows external services to discover and communicate with
  K8S services.

`
