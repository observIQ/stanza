package input

import (
	"context"
	"fmt"
	"sync"
	"time"

	backoff "github.com/cenkalti/backoff/v4"
	"github.com/observiq/stanza/entry"
	"github.com/observiq/stanza/errors"
	"github.com/observiq/stanza/operator"
	"github.com/observiq/stanza/operator/helper"
	"go.uber.org/zap"
	apiv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	watch "k8s.io/apimachinery/pkg/watch"
	corev1 "k8s.io/client-go/kubernetes/typed/core/v1"
	"k8s.io/client-go/rest"
)

func init() {
	operator.Register("k8s_event_input", func() operator.Builder { return NewK8sEventsConfig("") })
}

// NewK8sEventsConfig creates a default K8sEventsConfig
func NewK8sEventsConfig(operatorID string) *K8sEventsConfig {
	return &K8sEventsConfig{
		InputConfig: helper.NewInputConfig(operatorID, "k8s_event_input"),
	}
}

// K8sEventsConfig is the configuration of K8sEvents operator
type K8sEventsConfig struct {
	helper.InputConfig `yaml:",inline"`
	Namespaces         []string
}

// Build will build a k8s_event_input operator from the supplied configuration
func (c K8sEventsConfig) Build(context operator.BuildContext) (operator.Operator, error) {
	input, err := c.InputConfig.Build(context)
	if err != nil {
		return nil, errors.Wrap(err, "build transformer")
	}

	return &K8sEvents{
		InputOperator: input,
		namespaces:    c.Namespaces,
	}, nil
}

// K8sEvents is an operator for generating logs from k8s events
type K8sEvents struct {
	helper.InputOperator
	client     corev1.CoreV1Interface
	namespaces []string

	cancel func()
	wg     sync.WaitGroup
}

// Start implements the operator.Operator interface
func (k *K8sEvents) Start() error {
	ctx, cancel := context.WithCancel(context.Background())
	k.cancel = cancel

	// Currently, we only support running in the cluster. In contrast to the
	// k8s_metadata_decorator, it may make sense to relax this restriction
	// by exposing client config options.
	config, err := rest.InClusterConfig()
	if err != nil {
		return errors.NewError(
			"agent not in kubernetes cluster",
			"the k8s_event_input operator only supports running in a pod inside a kubernetes cluster",
		)
	}

	k.client, err = corev1.NewForConfig(config)
	if err != nil {
		return errors.Wrap(err, "build client")
	}

	// Get all namespaces if left empty
	if len(k.namespaces) == 0 {
		k.namespaces, err = listNamespaces(ctx, k.client)
		if err != nil {
			return errors.Wrap(err, "list namespaces")
		}
	}

	// Test connection
	testWatcher, err := k.client.Events(k.namespaces[0]).Watch(ctx, metav1.ListOptions{})
	if err != nil {
		return fmt.Errorf("test connection: list events for namespace '%s': %s", k.namespaces[0], err)
	}
	testWatcher.Stop()

	for _, ns := range k.namespaces {
		k.startWatchingNamespace(ctx, ns)
	}

	return nil
}

// Stop implements operator.Operator
func (k *K8sEvents) Stop() error {
	k.cancel()
	k.wg.Wait()
	return nil
}

// listNamespaces gets a full list of namespaces from the client
func listNamespaces(ctx context.Context, client corev1.CoreV1Interface) ([]string, error) {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	res, err := client.Namespaces().List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}
	namespaces := make([]string, 0, 10)
	for _, ns := range res.Items {
		namespaces = append(namespaces, ns.Name)
	}
	return namespaces, nil
}

// startWatchingNamespace creates a goroutine that watches the events for a
// specific namespace
func (k *K8sEvents) startWatchingNamespace(ctx context.Context, ns string) {
	k.wg.Add(1)
	go func() {
		defer k.wg.Done()

		b := backoff.NewExponentialBackOff()
		for {
			select {
			case <-ctx.Done():
				return
			case <-time.After(b.NextBackOff()):
			}

			watcher, err := k.client.Events(ns).Watch(ctx, metav1.ListOptions{})
			if err != nil {
				k.Errorw("Failed to start watcher", zap.Error(err))
				continue
			}
			b.Reset()

			k.consumeWatchEvents(ctx, watcher.ResultChan())
		}
	}()
}

// consumeWatchEvents will read events from the watcher channel until the channel is closed
// or the context is canceled
func (k *K8sEvents) consumeWatchEvents(ctx context.Context, events <-chan watch.Event) {
	for {
		select {
		case event, ok := <-events:
			if !ok {
				k.Error("Watcher channel closed")
				return
			}

			typedEvent := event.Object.(*apiv1.Event)
			record, err := runtime.DefaultUnstructuredConverter.ToUnstructured(event.Object)
			if err != nil {
				k.Error("Failed to convert event to map", zap.Error(err))
				continue
			}

			entry, err := k.NewEntry(record)
			if err != nil {
				k.Error("Failed to create new entry from record", zap.Error(err))
				continue
			}

			entry.Timestamp = typedEvent.LastTimestamp.Time
			entry.AddLabel("event_type", string(event.Type))
			k.populateResource(typedEvent, entry)
			k.Write(ctx, entry)
		case <-ctx.Done():
			return
		}
	}
}

// populateResource uses the keys from Event.ObjectMeta to populate the resource of the entry
func (k *K8sEvents) populateResource(event *apiv1.Event, entry *entry.Entry) {
	entry.AddResourceKey("k8s.cluster.name", event.ClusterName)
	entry.AddResourceKey("k8s.namespace.name", event.Namespace)
	switch event.Kind {
	case "Pod":
		entry.AddResourceKey("k8s.pod.uid", string(event.UID))
		entry.AddResourceKey("k8s.pod.name", event.Name)
	case "Container":
		entry.AddResourceKey("k8s.container.name", event.Name)
	case "ReplicaSet":
		entry.AddResourceKey("k8s.replicaset.uid", string(event.UID))
		entry.AddResourceKey("k8s.replicaset.name", event.Name)
	case "Deployment":
		entry.AddResourceKey("k8s.deployment.uid", string(event.UID))
		entry.AddResourceKey("k8s.deployment.name", event.Name)
	case "StatefulSet":
		entry.AddResourceKey("k8s.statefulset.uid", string(event.UID))
		entry.AddResourceKey("k8s.statefulset.name", event.Name)
	case "DaemonSet":
		entry.AddResourceKey("k8s.daemonset.uid", string(event.UID))
		entry.AddResourceKey("k8s.daemonset.name", event.Name)
	case "Job":
		entry.AddResourceKey("k8s.job.uid", string(event.UID))
		entry.AddResourceKey("k8s.job.name", event.Name)
	case "CronJob":
		entry.AddResourceKey("k8s.cronjob.uid", string(event.UID))
		entry.AddResourceKey("k8s.cronjob.name", event.Name)
	}
}
