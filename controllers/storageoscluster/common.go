package storageoscluster

import (
	corev1 "k8s.io/api/core/v1"
)

const (
	// TaintNodeOutOfDisk will be added when node runs out of disk space, and
	// removed when disk space becomes available.
	// NOTE: It is not an upstream constant.
	TaintNodeOutOfDisk = "node.kubernetes.io/out-of-disk"
)

// getDefaultTolerations returns a collection of default tolerations for
// StorageOS related resources.
// NOTE: An empty effect matches all effects with the given key.
func getDefaultTolerations() []corev1.Toleration {
	return []corev1.Toleration{
		{
			Key:      corev1.TaintNodeDiskPressure,
			Operator: corev1.TolerationOpExists,
			Effect:   "",
		},
		{
			Key:      corev1.TaintNodeMemoryPressure,
			Operator: corev1.TolerationOpExists,
			Effect:   "",
		},
		{
			Key:      corev1.TaintNodeNetworkUnavailable,
			Operator: corev1.TolerationOpExists,
			Effect:   "",
		},
		{
			Key:      corev1.TaintNodeNotReady,
			Operator: corev1.TolerationOpExists,
			Effect:   "",
		},
		{
			Key:      TaintNodeOutOfDisk,
			Operator: corev1.TolerationOpExists,
			Effect:   "",
		},
		{
			Key:      corev1.TaintNodePIDPressure,
			Operator: corev1.TolerationOpExists,
			Effect:   "",
		},
		{
			Key:      corev1.TaintNodeUnreachable,
			Operator: corev1.TolerationOpExists,
			Effect:   "",
		},
		{
			Key:      corev1.TaintNodeUnschedulable,
			Operator: corev1.TolerationOpExists,
			Effect:   "",
		},
	}
}
