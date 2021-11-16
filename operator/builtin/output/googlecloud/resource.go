package googlecloud

import (
	"github.com/observiq/stanza/v2/entry"
	mrpb "google.golang.org/genproto/googleapis/api/monitoredres"
)

// For more about monitored resources, see:
// https://cloud.google.com/logging/docs/api/v2/resource-list#resource-types

func getResource(e *entry.Entry) *mrpb.MonitoredResource {
	rt := detectResourceType(e)
	if rt == "" {
		return nil
	}

	switch rt {
	case "k8s_pod":
		return k8sPodResource(e)
	case "k8s_container":
		return k8sContainerResource(e)
	case "k8s_node":
		return k8sNodeResource(e)
	case "k8s_cluster":
		return k8sClusterResource(e)
	case "generic_node":
		return genericNodeResource(e)
	}

	return nil
}

func detectResourceType(e *entry.Entry) string {
	if hasResource("k8s.pod.name", e) {
		if hasResource("container.name", e) {
			return "k8s_container"
		}
		if hasResource("k8s.container.name", e) {
			e.Resource["container.name"] = e.Resource["k8s.container.name"]
			delete(e.Resource, "k8s.container.name")
			return "k8s_container"
		}
		return "k8s_pod"
	}

	if hasResource("k8s.cluster.name", e) {
		if hasResource("host.name", e) {
			return "k8s_node"
		}
		return "k8s_cluster"
	}

	if hasResource("host.name", e) {
		return "generic_node"
	}

	return ""
}

func hasResource(key string, e *entry.Entry) bool {
	_, ok := e.Resource[key]
	return ok
}

func k8sPodResource(e *entry.Entry) *mrpb.MonitoredResource {
	m := &mrpb.MonitoredResource{
		Type: "k8s_pod",
		Labels: map[string]string{
			"pod_name":       e.Resource["k8s.pod.name"],
			"namespace_name": e.Resource["k8s.namespace.name"],
			"cluster_name":   e.Resource["k8s.cluster.name"],
			// TODO project id
		},
	}

	delete(e.Resource, "k8s.pod.name")
	delete(e.Resource, "k8s.namespace.name")
	delete(e.Resource, "k8s.cluster.name")

	return m
}

func k8sContainerResource(e *entry.Entry) *mrpb.MonitoredResource {
	m := &mrpb.MonitoredResource{
		Type: "k8s_container",
		Labels: map[string]string{
			"container_name": e.Resource["container.name"],
			"pod_name":       e.Resource["k8s.pod.name"],
			"namespace_name": e.Resource["k8s.namespace.name"],
			"cluster_name":   e.Resource["k8s.cluster.name"],
			// TODO project id
		},
	}

	delete(e.Resource, "container.name")
	delete(e.Resource, "k8s.pod.name")
	delete(e.Resource, "k8s.namespace.name")
	delete(e.Resource, "k8s.cluster.name")

	return m
}

func k8sNodeResource(e *entry.Entry) *mrpb.MonitoredResource {
	m := &mrpb.MonitoredResource{
		Type: "k8s_node",
		Labels: map[string]string{
			"cluster_name": e.Resource["k8s.cluster.name"],
			"node_name":    e.Resource["host.name"],
			// TODO project id
		},
	}

	delete(e.Resource, "k8s.cluster.name")
	delete(e.Resource, "host.name")

	return m
}

func k8sClusterResource(e *entry.Entry) *mrpb.MonitoredResource {
	m := &mrpb.MonitoredResource{
		Type: "k8s_cluster",
		Labels: map[string]string{
			"cluster_name": e.Resource["k8s.cluster.name"],
			// TODO project id
		},
	}

	delete(e.Resource, "k8s.cluster.name")

	return m
}

func genericNodeResource(e *entry.Entry) *mrpb.MonitoredResource {
	m := &mrpb.MonitoredResource{
		Type: "generic_node",
		Labels: map[string]string{
			"node_id": e.Resource["host.name"],
			// TODO project id
		},
	}

	delete(e.Resource, "host.name")

	return m
}
