package transformer

import (
	"context"
	"sync"
	"time"

	"github.com/bluemedora/bplogagent/entry"
	"github.com/bluemedora/bplogagent/errors"
	"github.com/bluemedora/bplogagent/plugin"
	"github.com/bluemedora/bplogagent/plugin/helper"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	corev1 "k8s.io/client-go/kubernetes/typed/core/v1"
	"k8s.io/client-go/rest"
)

func init() {
	plugin.Register("k8s_metadata_decorator", &K8sMetadataDecoratorConfig{})
}

type K8sMetadataDecoratorConfig struct {
	helper.TransformerConfig `yaml:",inline"`
	PodNameField             *entry.Field    `json:"pod_name_field,omitempty"  yaml:"pod_name_field,omitempty"`
	NamespaceField           *entry.Field    `json:"namespace_field,omitempty" yaml:"namespace_field,omitempty"`
	CacheTTL                 plugin.Duration `json:"cache_ttl,omitempty"       yaml:"cache_ttl,omitempty"`
}

func (c K8sMetadataDecoratorConfig) Build(context plugin.BuildContext) (plugin.Plugin, error) {
	transformer, err := c.TransformerConfig.Build(context)
	if err != nil {
		return nil, errors.Wrap(err, "build transformer")
	}

	if c.PodNameField == nil {
		field := entry.NewField("pod_name")
		c.PodNameField = &field
	}

	if c.NamespaceField == nil {
		field := entry.NewField("namespace")
		c.NamespaceField = &field
	}

	config, err := rest.InClusterConfig()
	if err != nil {
		return nil, errors.NewError(
			"agent not in kubernetes cluster",
			"the k8s_metadata_decorator plugin only supports running in a pod inside a kubernetes cluster",
		)
	}

	if c.CacheTTL.Duration == time.Duration(0) {
		c.CacheTTL.Duration = 10 * time.Minute
	}

	return &K8sMetadataDecorator{
		TransformerPlugin: transformer,
		clientConfig:      config,
		podNameField:      *c.PodNameField,
		namespaceField:    *c.NamespaceField,
		cache_ttl:         time.Hour,
	}, nil
}

type K8sMetadataDecorator struct {
	helper.TransformerPlugin
	podNameField   entry.Field
	namespaceField entry.Field

	clientConfig *rest.Config
	client       *corev1.CoreV1Client

	namespaceCache sync.Map
	podCache       sync.Map
	cache_ttl      time.Duration
}

type MetadataCacheEntry struct {
	ExpirationTime time.Time
	Labels         map[string]string
	Annotations    map[string]string
}

func (c *K8sMetadataDecorator) Start() error {
	var err error
	c.client, err = corev1.NewForConfig(c.clientConfig)
	if err != nil {
		return errors.Wrap(err, "build client")
	}

	// Test connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	namespaceList, err := c.client.Namespaces().List(ctx, metav1.ListOptions{})
	if err != nil {
		return errors.Wrap(err, "test connection list namespaces")
	}

	if len(namespaceList.Items) == 0 {
		c.Warn("During test connection, namespace list came back empty")
		return nil
	}

	namespaceName := namespaceList.Items[0].ObjectMeta.Name
	_, err = c.client.Pods(namespaceName).List(ctx, metav1.ListOptions{})
	if err != nil {
		return errors.Wrap(err, "test connection list pods")
	}

	return nil
}

func (c *K8sMetadataDecorator) Process(ctx context.Context, entry *entry.Entry) error {
	var podName string
	err := entry.Read(c.podNameField, &podName)
	if err != nil {
		return c.HandleEntryError(ctx, entry, errors.Wrap(err, "find pod name").WithDetails("search_field", c.podNameField.String()))
	}

	var namespace string
	err = entry.Read(c.namespaceField, &namespace)
	if err != nil {
		return c.HandleEntryError(ctx, entry, errors.Wrap(err, "find namespace").WithDetails("search_field", c.podNameField.String()))
	}

	nsMeta, err := c.getNamespaceMetadata(ctx, namespace)
	if err != nil {
		return c.HandleEntryError(ctx, entry, err)
	}
	c.decorateEntryWithNamespaceMetadata(nsMeta, entry)

	podMeta, err := c.getPodMetadata(ctx, namespace, podName)
	if err != nil {
		return c.HandleEntryError(ctx, entry, err)
	}
	c.decorateEntryWithPodMetadata(podMeta, entry)

	return c.Output.Process(ctx, entry)
}

func (c *K8sMetadataDecorator) getNamespaceMetadata(ctx context.Context, namespace string) (MetadataCacheEntry, error) {
	cacheEntry, ok := c.namespaceCache.Load(namespace)
	now := time.Now()
	if !ok || cacheEntry.(MetadataCacheEntry).ExpirationTime.Before(now) {
		ctx, cancel := context.WithTimeout(ctx, 5*time.Second) // TODO configurable timeout
		defer cancel()
		namespaceResponse, err := c.client.Namespaces().Get(ctx, namespace, metav1.GetOptions{})
		if err != nil {
			cacheEntry = MetadataCacheEntry{ExpirationTime: now.Add(10 * time.Second)}
			c.namespaceCache.Store(namespace, cacheEntry)
			return cacheEntry.(MetadataCacheEntry), errors.Wrap(err, "get namespace metadata").WithDetails("namespace", namespace).WithDetails("retry_after", "10s")
		}
		cacheEntry = MetadataCacheEntry{
			ExpirationTime: now.Add(c.cache_ttl),
			Labels:         namespaceResponse.Labels,
			Annotations:    namespaceResponse.Annotations,
		}
		c.namespaceCache.Store(namespace, cacheEntry)
	}
	return cacheEntry.(MetadataCacheEntry), nil
}

func (c *K8sMetadataDecorator) getPodMetadata(ctx context.Context, namespace, podName string) (MetadataCacheEntry, error) {
	key := namespace + ":" + podName
	cacheEntry, ok := c.podCache.Load(key)

	now := time.Now()
	if !ok || cacheEntry.(MetadataCacheEntry).ExpirationTime.Before(now) {
		ctx, cancel := context.WithTimeout(ctx, 5*time.Second) // TODO configurable timeout
		defer cancel()
		podResponse, err := c.client.Pods(namespace).Get(ctx, podName, metav1.GetOptions{})
		if err != nil {
			cacheEntry = MetadataCacheEntry{ExpirationTime: now.Add(10 * time.Second)}
			c.podCache.Store(key, cacheEntry)
			return cacheEntry.(MetadataCacheEntry), errors.Wrap(err, "get pod metadata").WithDetails(
				"namespace", namespace,
				"pod_name", podName,
				"retry_after", "10s",
			)
		}
		cacheEntry = MetadataCacheEntry{
			ExpirationTime: now.Add(c.cache_ttl),
			Labels:         podResponse.Labels,
			Annotations:    podResponse.Annotations,
		}
		c.podCache.Store(key, cacheEntry)
	}
	return cacheEntry.(MetadataCacheEntry), nil
}

func (c *K8sMetadataDecorator) decorateEntryWithNamespaceMetadata(nsMeta MetadataCacheEntry, entry *entry.Entry) {
	for k, v := range nsMeta.Labels {
		entry.Labels["k8s_ns_annotation/"+k] = v
	}

	for k, v := range nsMeta.Labels {
		entry.Labels["k8s_ns_label/"+k] = v
	}
}

func (c *K8sMetadataDecorator) decorateEntryWithPodMetadata(nsMeta MetadataCacheEntry, entry *entry.Entry) {
	for k, v := range nsMeta.Labels {
		entry.Labels["k8s_pod_annotation/"+k] = v
	}

	for k, v := range nsMeta.Labels {
		entry.Labels["k8s_pod_label/"+k] = v
	}
}
