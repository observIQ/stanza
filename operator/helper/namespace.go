package helper

import "fmt"

// CanNamespace will return a boolean indicating if an id can be namespaced.
func CanNamespace(id string, exclusions []string) bool {
	for _, key := range exclusions {
		if key == id {
			return false
		}
	}
	return true
}

// AddNamespace will add a namespace to an id.
func AddNamespace(id string, namespace string) string {
	return fmt.Sprintf("%s.%s", namespace, id)
}
