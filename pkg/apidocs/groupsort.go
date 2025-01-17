package apidocs

import (
	"sort"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// Custom sort function to prioritize groups like apps/v1 and v1 at the top
func customSortGroups(groups []*metav1.APIResourceList) {
	// Prioritize these groups at the top
	topLevels := []string{
		"v1",
		"apps/v1",
		// "batch/v1",
		// "rbac.authorization.k8s.io/v1",
		// "networking.k8s.io/v1",
		// "gateway.networking.k8s.io/v1",
		// "gateway.networking.k8s.io/v1beta1",
	}

	sort.SliceStable(groups, func(i, j int) bool {
		for _, t := range topLevels {
			if groups[i].GroupVersion == t {
				return true
			}
			if groups[j].GroupVersion == t {
				return false
			}
		}

		// Default alphabetical sorting
		return groups[i].GroupVersion < groups[j].GroupVersion
	})
}
