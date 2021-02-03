package transform

import (
	corev1 "k8s.io/api/core/v1"
	kyaml "sigs.k8s.io/kustomize/kyaml/yaml"
)

// createSecretEnvVarSource creates a secret env var source kyaml RNode.
func createSecretEnvVarSource(secretName, secretKey string) (*kyaml.RNode, error) {
	envVarSrc := corev1.EnvVarSource{
		SecretKeyRef: &corev1.SecretKeySelector{
			LocalObjectReference: corev1.LocalObjectReference{Name: secretName},
			Key:                  secretKey,
		},
	}
	return goToRNode(envVarSrc)
}

// createValueFromFieldEnvVarSource creates a value from field env var source
// kyaml RNode.
func createValueFromFieldEnvVarSource(fieldPath string) (*kyaml.RNode, error) {
	envVarSrc := corev1.EnvVarSource{
		FieldRef: &corev1.ObjectFieldSelector{
			APIVersion: "v1",
			FieldPath:  fieldPath,
		},
	}
	return goToRNode(envVarSrc)
}
