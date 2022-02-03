package googlecloud

import (
	"testing"

	"github.com/open-telemetry/opentelemetry-log-collection/entry"
	"github.com/stretchr/testify/require"
	"google.golang.org/genproto/googleapis/api/monitoredres"
)

func TestGetResource(t *testing.T) {
	testCases := []struct {
		name     string
		entry    *entry.Entry
		resource *monitoredres.MonitoredResource
	}{
		{
			name: "k8s pod",
			entry: &entry.Entry{
				Resource: map[string]string{
					"k8s.pod.name":       "test_pod",
					"k8s.namespace.name": "test_namespace",
					"k8s.cluster.name":   "test_cluster",
				},
			},
			resource: &monitoredres.MonitoredResource{
				Type: k8sPod,
				Labels: map[string]string{
					"pod_name":       "test_pod",
					"namespace_name": "test_namespace",
					"cluster_name":   "test_cluster",
				},
			},
		},
		{
			name: "k8s container simple",
			entry: &entry.Entry{
				Resource: map[string]string{
					"container.name":     "test_container",
					"k8s.pod.name":       "test_pod",
					"k8s.namespace.name": "test_namespace",
					"k8s.cluster.name":   "test_cluster",
				},
			},
			resource: &monitoredres.MonitoredResource{
				Type: k8sContainer,
				Labels: map[string]string{
					"container_name": "test_container",
					"pod_name":       "test_pod",
					"namespace_name": "test_namespace",
					"cluster_name":   "test_cluster",
				},
			},
		},
		{
			name: "k8s container longhand",
			entry: &entry.Entry{
				Resource: map[string]string{
					"k8s.container.name": "test_container",
					"k8s.pod.name":       "test_pod",
					"k8s.namespace.name": "test_namespace",
					"k8s.cluster.name":   "test_cluster",
				},
			},
			resource: &monitoredres.MonitoredResource{
				Type: k8sContainer,
				Labels: map[string]string{
					"container_name": "test_container",
					"pod_name":       "test_pod",
					"namespace_name": "test_namespace",
					"cluster_name":   "test_cluster",
				},
			},
		},
		{
			name: "k8s node",
			entry: &entry.Entry{
				Resource: map[string]string{
					"k8s.cluster.name": "test_cluster",
					"host.name":        "test_host",
				},
			},
			resource: &monitoredres.MonitoredResource{
				Type: k8sNode,
				Labels: map[string]string{
					"cluster_name": "test_cluster",
					"node_name":    "test_host",
				},
			},
		},
		{
			name: "k8s cluster",
			entry: &entry.Entry{
				Resource: map[string]string{
					"k8s.cluster.name": "test_cluster",
				},
			},
			resource: &monitoredres.MonitoredResource{
				Type: k8sCluster,
				Labels: map[string]string{
					"cluster_name": "test_cluster",
				},
			},
		},
		{
			name: "unknown",
			entry: &entry.Entry{
				Resource: map[string]string{},
			},
			resource: nil,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := createResource(tc.entry)
			require.Equal(t, tc.resource, result)
		})
	}
}
