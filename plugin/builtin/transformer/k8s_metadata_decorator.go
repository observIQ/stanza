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

	namespaceCache MetadataCache
	podCache       MetadataCache
	cache_ttl      time.Duration
}

type MetadataCacheEntry struct {
	ExpirationTime time.Time
	Labels         map[string]string
	Annotations    map[string]string
}

type MetadataCache struct {
	m sync.Map
}

func (m *MetadataCache) Load(key string) (MetadataCacheEntry, bool) {
	entry, ok := m.m.Load(key)
	return entry.(MetadataCacheEntry), ok
}

func (m *MetadataCache) Store(key string, entry MetadataCacheEntry) {
	m.m.Store(key, entry)
}

func (k *K8sMetadataDecorator) Start() error {
	var err error
	k.client, err = corev1.NewForConfig(k.clientConfig)
	if err != nil {
		return errors.Wrap(err, "build client")
	}

	// Test connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	namespaceList, err := k.client.Namespaces().List(ctx, metav1.ListOptions{})
	if err != nil {
		return errors.Wrap(err, "test connection list namespaces")
	}

	if len(namespaceList.Items) == 0 {
		k.Warn("During test connection, namespace list came back empty")
		return nil
	}

	namespaceName := namespaceList.Items[0].ObjectMeta.Name
	_, err = k.client.Pods(namespaceName).List(ctx, metav1.ListOptions{})
	if err != nil {
		return errors.Wrap(err, "test connection list pods")
	}

	return nil
}

func (k *K8sMetadataDecorator) Process(ctx context.Context, entry *entry.Entry) error {
	var podName string
	err := entry.Read(k.podNameField, &podName)
	if err != nil {
		return k.HandleEntryError(ctx, entry, errors.Wrap(err, "find pod name").WithDetails("search_field", k.podNameField.String()))
	}

	var namespace string
	err = entry.Read(k.namespaceField, &namespace)
	if err != nil {
		return k.HandleEntryError(ctx, entry, errors.Wrap(err, "find namespace").WithDetails("search_field", k.podNameField.String()))
	}

	nsMeta, err := k.getNamespaceMetadata(ctx, namespace)
	if err != nil {
		return k.HandleEntryError(ctx, entry, err)
	}
	k.decorateEntryWithNamespaceMetadata(nsMeta, entry)

	podMeta, err := k.getPodMetadata(ctx, namespace, podName)
	if err != nil {
		return k.HandleEntryError(ctx, entry, err)
	}
	k.decorateEntryWithPodMetadata(podMeta, entry)

	k.Write(ctx, entry)
	return nil
}

func (k *K8sMetadataDecorator) getNamespaceMetadata(ctx context.Context, namespace string) (MetadataCacheEntry, error) {
	cacheEntry, ok := k.namespaceCache.Load(namespace)

	var err error
	if !ok || cacheEntry.ExpirationTime.Before(time.Now()) {
		cacheEntry, err = k.refreshNamespaceMetadata(ctx, namespace)
	}

	return cacheEntry, err
}

func (k *K8sMetadataDecorator) getPodMetadata(ctx context.Context, namespace, podName string) (MetadataCacheEntry, error) {
	key := namespace + ":" + podName
	cacheEntry, ok := k.podCache.Load(key)

	var err error
	if !ok || cacheEntry.ExpirationTime.Before(time.Now()) {
		cacheEntry, err = k.refreshPodMetadata(ctx, namespace, podName)
	}

	return cacheEntry, err
}

func (k *K8sMetadataDecorator) refreshNamespaceMetadata(ctx context.Context, namespace string) (MetadataCacheEntry, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	// Query the API
	namespaceResponse, err := k.client.Namespaces().Get(ctx, namespace, metav1.GetOptions{})
	if err != nil {
		// Add an empty entry to the cache so we don't continuously retry
		cacheEntry := MetadataCacheEntry{ExpirationTime: time.Now().Add(10 * time.Second)}
		k.namespaceCache.Store(namespace, cacheEntry)
		return cacheEntry, errors.Wrap(err, "get namespace metadata").WithDetails("namespace", namespace).WithDetails("retry_after", "10s")
	}

	// Cache the results
	cacheEntry := MetadataCacheEntry{
		ExpirationTime: time.Now().Add(k.cache_ttl),
		Labels:         namespaceResponse.Labels,
		Annotations:    namespaceResponse.Annotations,
	}
	k.namespaceCache.Store(namespace, cacheEntry)

	return cacheEntry, nil
}

func (k *K8sMetadataDecorator) refreshPodMetadata(ctx context.Context, namespace, podName string) (MetadataCacheEntry, error) {
	key := namespace + ":" + podName

	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	// Query the API
	podResponse, err := k.client.Pods(namespace).Get(ctx, podName, metav1.GetOptions{})
	if err != nil {
		// Add an empty entry to the cache so we don't continuously retry
		cacheEntry := MetadataCacheEntry{ExpirationTime: time.Now().Add(10 * time.Second)}
		k.podCache.Store(key, cacheEntry)

		return cacheEntry, errors.Wrap(err, "get pod metadata").WithDetails(
			"namespace", namespace,
			"pod_name", podName,
			"retry_after", "10s",
		)
	}

	// Cache the results
	cacheEntry := MetadataCacheEntry{
		ExpirationTime: time.Now().Add(k.cache_ttl),
		Labels:         podResponse.Labels,
		Annotations:    podResponse.Annotations,
	}
	k.podCache.Store(key, cacheEntry)

	return cacheEntry, nil
}

func (k *K8sMetadataDecorator) decorateEntryWithNamespaceMetadata(nsMeta MetadataCacheEntry, entry *entry.Entry) {
	for k, v := range nsMeta.Labels {
		entry.Labels["k8s_ns_annotation/"+k] = v
	}

	for k, v := range nsMeta.Labels {
		entry.Labels["k8s_ns_label/"+k] = v
	}
}

func (k *K8sMetadataDecorator) decorateEntryWithPodMetadata(nsMeta MetadataCacheEntry, entry *entry.Entry) {
	for k, v := range nsMeta.Labels {
		entry.Labels["k8s_pod_annotation/"+k] = v
	}

	for k, v := range nsMeta.Labels {
		entry.Labels["k8s_pod_label/"+k] = v
	}
}
