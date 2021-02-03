package transform

import (
	corev1 "k8s.io/api/core/v1"
	kyaml "sigs.k8s.io/kustomize/kyaml/yaml"
)

const (
	// Volume source names.
	volSrcHostPath  = "hostPath"
	volSrcConfigMap = "configMap"
	volSrcSecret    = "secret"
)

// createHostPathVolumeSource creates a host path volume source kyaml RNode.
func createHostPathVolumeSource(path string, pathType *corev1.HostPathType) (*kyaml.RNode, error) {
	hostPath := &corev1.HostPathVolumeSource{
		Path: path,
		Type: pathType,
	}
	return goToRNode(hostPath)
}

// createConfigMapVolumeSource creates a configmap volume source kyaml RNode.
func createConfigMapVolumeSource(nameVal string, keyToPaths []corev1.KeyToPath) (*kyaml.RNode, error) {
	configmapVS := &corev1.ConfigMapVolumeSource{
		LocalObjectReference: corev1.LocalObjectReference{Name: nameVal},
		Items:                keyToPaths,
	}
	return goToRNode(configmapVS)
}

// createSecretVolumeSource creates a secret volume source kyaml RNode.
func createSecretVolumeSource(nameVal string, keyToPaths []corev1.KeyToPath) (*kyaml.RNode, error) {
	secretVS := &corev1.SecretVolumeSource{
		SecretName: nameVal,
		Items:      keyToPaths,
	}
	return goToRNode(secretVS)
}
