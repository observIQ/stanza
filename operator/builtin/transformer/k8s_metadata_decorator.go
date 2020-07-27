package transformer

import (
	"context"
	"sync"
	"time"

	"github.com/observiq/carbon/entry"
	"github.com/observiq/carbon/errors"
	"github.com/observiq/carbon/operator"
	"github.com/observiq/carbon/operator/helper"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	corev1 "k8s.io/client-go/kubernetes/typed/core/v1"
	"k8s.io/client-go/rest"
)

func init() {
	operator.Register("k8s_metadata_decorator", func() operator.Builder { return NewK8smetadataDecoratorConfig("") })
}

func NewK8smetadataDecoratorConfig(operatorID string) *K8sMetadataDecoratorConfig {
	return &K8sMetadataDecoratorConfig{
		TransformerConfig: helper.NewTransformerConfig(operatorID, "k8s_metadata_decorator"),
		PodNameField:      entry.NewRecordField("pod_name"),
		NamespaceField:    entry.NewRecordField("namespace"),
		CacheTTL:          operator.Duration{Duration: 10 * time.Minute},
		Timeout:           operator.Duration{Duration: 10 * time.Second},
	}
}

// K8sMetadataDecoratorConfig is the configuration of k8s_metadata_decorator operator
type K8sMetadataDecoratorConfig struct {
	helper.TransformerConfig `yaml:",inline"`
	PodNameField             entry.Field       `json:"pod_name_field,omitempty"  yaml:"pod_name_field,omitempty"`
	NamespaceField           entry.Field       `json:"namespace_field,omitempty" yaml:"namespace_field,omitempty"`
	CacheTTL                 operator.Duration `json:"cache_ttl,omitempty"       yaml:"cache_ttl,omitempty"`
	Timeout                  operator.Duration `json:"timeout,omitempty"         yaml:"timeout,omitempty"`
}

// Build will build a k8s_metadata_decorator operator from the supplied configuration
func (c K8sMetadataDecoratorConfig) Build(context operator.BuildContext) (operator.Operator, error) {
	transformer, err := c.TransformerConfig.Build(context)
	if err != nil {
		return nil, errors.Wrap(err, "build transformer")
	}

	return &K8sMetadataDecorator{
		TransformerOperator: transformer,
		podNameField:        c.PodNameField,
		namespaceField:      c.NamespaceField,
		cacheTTL:            c.CacheTTL.Raw(),
		timeout:             c.Timeout.Raw(),
	}, nil
}

// K8sMetadataDecorator is an operator for decorating entries with kubernetes metadata
type K8sMetadataDecorator struct {
	helper.TransformerOperator
	podNameField   entry.Field
	namespaceField entry.Field

	client *corev1.CoreV1Client

	namespaceCache MetadataCache
	podCache       MetadataCache
	cacheTTL       time.Duration
	timeout        time.Duration
}

// MetadataCacheEntry is an entry in the metadata cache
type MetadataCacheEntry struct {
	ExpirationTime time.Time
	Labels         map[string]string
	Annotations    map[string]string
}

// MetadataCache is a cache of kubernetes metadata
type MetadataCache struct {
	m sync.Map
}

// Load will return an entry stored in the metadata cache
func (m *MetadataCache) Load(key string) (MetadataCacheEntry, bool) {
	entry, ok := m.m.Load(key)
	if ok {
		return entry.(MetadataCacheEntry), ok
	}
	return MetadataCacheEntry{}, ok
}

// Store will store an entry in the metadata cache
func (m *MetadataCache) Store(key string, entry MetadataCacheEntry) {
	m.m.Store(key, entry)
}

// Start will start the k8s_metadata_decorator operator
func (k *K8sMetadataDecorator) Start() error {
	config, err := rest.InClusterConfig()
	if err != nil {
		return errors.NewError(
			"agent not in kubernetes cluster",
			"the k8s_metadata_decorator operator only supports running in a pod inside a kubernetes cluster",
		)
	}

	k.client, err = corev1.NewForConfig(config)
	if err != nil {
		return errors.Wrap(err, "build client")
	}

	// Test connection
	ctx, cancel := context.WithTimeout(context.Background(), k.timeout)
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

// Process will process an entry received by the k8s_metadata_decorator operator
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
	ctx, cancel := context.WithTimeout(ctx, k.timeout)
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
		ExpirationTime: time.Now().Add(k.cacheTTL),
		Labels:         namespaceResponse.Labels,
		Annotations:    namespaceResponse.Annotations,
	}
	k.namespaceCache.Store(namespace, cacheEntry)

	return cacheEntry, nil
}

func (k *K8sMetadataDecorator) refreshPodMetadata(ctx context.Context, namespace, podName string) (MetadataCacheEntry, error) {
	key := namespace + ":" + podName

	ctx, cancel := context.WithTimeout(ctx, k.timeout)
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
		ExpirationTime: time.Now().Add(k.cacheTTL),
		Labels:         podResponse.Labels,
		Annotations:    podResponse.Annotations,
	}
	k.podCache.Store(key, cacheEntry)

	return cacheEntry, nil
}

func (k *K8sMetadataDecorator) decorateEntryWithNamespaceMetadata(nsMeta MetadataCacheEntry, entry *entry.Entry) {
	if entry.Labels == nil {
		entry.Labels = make(map[string]string)
	}

	for k, v := range nsMeta.Annotations {
		entry.Labels["k8s_ns_annotation/"+k] = v
	}

	for k, v := range nsMeta.Labels {
		entry.Labels["k8s_ns_label/"+k] = v
	}
}

func (k *K8sMetadataDecorator) decorateEntryWithPodMetadata(nsMeta MetadataCacheEntry, entry *entry.Entry) {
	if entry.Labels == nil {
		entry.Labels = make(map[string]string)
	}

	for k, v := range nsMeta.Annotations {
		entry.Labels["k8s_pod_annotation/"+k] = v
	}

	for k, v := range nsMeta.Labels {
		entry.Labels["k8s_pod_label/"+k] = v
	}
}
