package googlecloud

import (
	"github.com/open-telemetry/opentelemetry-log-collection/entry"
	"google.golang.org/genproto/googleapis/api/monitoredres"
)

// ResourceType is a monitored resource type
type ResourceType = string

const (
	k8sPod       ResourceType = "k8s_pod"
	k8sContainer ResourceType = "k8s_container"
	k8sNode      ResourceType = "k8s_node"
	k8sCluster   ResourceType = "k8s_cluster"
	genericNode  ResourceType = "generic_node"
)

// ResourceKey is a key used to distinguish a resource
type ResourceKey = string

const (
	containerName    ResourceKey = "container.name"
	k8sContainerName ResourceKey = "k8s.container.name"
	k8sPodName       ResourceKey = "k8s.pod.name"
	k8sClusterName   ResourceKey = "k8s.cluster.name"
	k8sNamespace     ResourceKey = "k8s.namespace.name"
	hostName         ResourceKey = "host.name"
)

// createResource creates a monitored resource from the supplied entry.
// For more about monitored resources, see:
// https://cloud.googlentry.com/logging/docs/api/v2/resource-list#resource-types
func createResource(entry *entry.Entry) *monitoredres.MonitoredResource {
	switch getResourceType(entry) {
	case k8sPod:
		return createPodResource(entry)
	case k8sContainer:
		return createContainerResource(entry)
	case k8sNode:
		return createNodeResource(entry)
	case k8sCluster:
		return createClusterResource(entry)
	case genericNode:
		return createGenericResource(entry)
	default:
		return nil
	}
}

// getResourceType returns the first resource type that matches the entry
func getResourceType(e *entry.Entry) ResourceType {
	switch {
	case hasResourceKeys(e, k8sPodName, containerName):
		return k8sContainer
	case hasResourceKeys(e, k8sPodName, k8sContainerName):
		return k8sContainer
	case hasResourceKeys(e, k8sPodName):
		return k8sPod
	case hasResourceKeys(e, k8sClusterName, hostName):
		return k8sNode
	case hasResourceKeys(e, k8sClusterName):
		return k8sCluster
	case hasResourceKeys(e, hostName):
		return genericNode
	default:
		return ""
	}
}

// hasResourceKeys returns true if an entry posesses all resource keys
func hasResourceKeys(entry *entry.Entry, keys ...string) bool {
	for _, key := range keys {
		if _, ok := entry.Resource[key]; !ok {
			return false
		}
	}

	return true
}

// createPodResource creates a pod resource from the entry
func createPodResource(entry *entry.Entry) *monitoredres.MonitoredResource {
	resource := &monitoredres.MonitoredResource{
		Type: k8sPod,
		Labels: map[string]string{
			"pod_name":       entry.Resource[k8sPodName],
			"namespace_name": entry.Resource[k8sNamespace],
			"cluster_name":   entry.Resource[k8sClusterName],
		},
	}

	delete(entry.Resource, k8sPodName)
	delete(entry.Resource, k8sNamespace)
	delete(entry.Resource, k8sClusterName)

	return resource
}

// createContainerResource creates a container resource from the entry
func createContainerResource(entry *entry.Entry) *monitoredres.MonitoredResource {
	containerName := entry.Resource[containerName]
	if containerName == "" {
		containerName = entry.Resource[k8sContainerName]
	}

	resource := &monitoredres.MonitoredResource{
		Type: k8sContainer,
		Labels: map[string]string{
			"container_name": containerName,
			"pod_name":       entry.Resource[k8sPodName],
			"namespace_name": entry.Resource[k8sNamespace],
			"cluster_name":   entry.Resource[k8sClusterName],
		},
	}

	delete(entry.Resource, containerName)
	delete(entry.Resource, k8sContainerName)
	delete(entry.Resource, k8sPodName)
	delete(entry.Resource, k8sNamespace)
	delete(entry.Resource, k8sClusterName)

	return resource
}

// createNodeResource creates a node resource from the entry
func createNodeResource(entry *entry.Entry) *monitoredres.MonitoredResource {
	resource := &monitoredres.MonitoredResource{
		Type: k8sNode,
		Labels: map[string]string{
			"cluster_name": entry.Resource[k8sClusterName],
			"node_name":    entry.Resource[hostName],
		},
	}

	delete(entry.Resource, k8sClusterName)
	delete(entry.Resource, hostName)

	return resource
}

// createClusterResource creates a cluster resource from the entry
func createClusterResource(entry *entry.Entry) *monitoredres.MonitoredResource {
	resource := &monitoredres.MonitoredResource{
		Type: k8sCluster,
		Labels: map[string]string{
			"cluster_name": entry.Resource[k8sClusterName],
		},
	}

	delete(entry.Resource, k8sClusterName)

	return resource
}

// createGenericResource creates a generic resource from the entry
func createGenericResource(entry *entry.Entry) *monitoredres.MonitoredResource {
	resource := &monitoredres.MonitoredResource{
		Type: genericNode,
		Labels: map[string]string{
			"node_id": entry.Resource[hostName],
		},
	}

	delete(entry.Resource, hostName)

	return resource
}
