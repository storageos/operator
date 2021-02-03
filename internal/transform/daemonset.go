package transform

import (
	"fmt"

	"github.com/darkowlzz/operator-toolkit/declarative/transform"
	corev1 "k8s.io/api/core/v1"
	kyaml "sigs.k8s.io/kustomize/kyaml/yaml"
)

// SetDaemonSetEnvVarFunc sets the environment variable in a DaemonSet
// container for the given key and value field.
func SetDaemonSetEnvVarFunc(container string, key string, valField string, value *kyaml.RNode) transform.TransformFunc {
	containerSelector := fmt.Sprintf("[name=%s]", container)
	envVarSelector := fmt.Sprintf("[name=%s]", key)
	path := []string{"spec", "template", "spec", "containers", containerSelector, "env", envVarSelector}
	return SetScalarNodeFunc(valField, value, path...)
}

// SetDaemonSetEnvVarStringFunc sets a string value environment variable for a
// given container in a DaemonSet.
func SetDaemonSetEnvVarStringFunc(container, key, val string) transform.TransformFunc {
	return SetDaemonSetEnvVarFunc(container, key, envVarValue, kyaml.NewScalarRNode(val))
}

// SetDaemonSetEnvVarValueFromSecretFunc sets a valueFrom secretKeyRef
// environment variable for a given container in a DaemonSet.
func SetDaemonSetEnvVarValueFromSecretFunc(container, key, secretName, secretKey string) (transform.TransformFunc, error) {
	// Construct secret env var source RNode.
	envVarSrc, err := createSecretEnvVarSource(secretName, secretKey)
	if err != nil {
		return nil, err
	}
	return SetDaemonSetEnvVarFunc(container, key, envVarValueFrom, envVarSrc), nil
}

// SetDaemonSetEnvVarValueFromFieldFunc sets a valueFrom fieldRef environment
// variable for a given container in a DaemonSet.
func SetDaemonSetEnvVarValueFromFieldFunc(container, key, fieldPath string) (transform.TransformFunc, error) {
	// Construct env var source from field ref RNode.
	envVarSrc, err := createValueFromFieldEnvVarSource(fieldPath)
	if err != nil {
		return nil, err
	}
	return SetDaemonSetEnvVarFunc(container, key, envVarValueFrom, envVarSrc), nil
}

// SetDaemonSetVolumeFunc sets a volume in a DaemonSet for the given name and
// volume source.
func SetDaemonSetVolumeFunc(volume string, volumeSource string, value *kyaml.RNode) transform.TransformFunc {
	volumeSelector := fmt.Sprintf("[name=%s]", volume)
	path := []string{"spec", "template", "spec", "volumes", volumeSelector}
	return SetScalarNodeFunc(volumeSource, value, path...)
}

// SetDaemonSetHostPathVolumeFunc sets a volume in a DaemonSet for a host path
// volume source.
func SetDaemonSetHostPathVolumeFunc(volume, path string, pathType *corev1.HostPathType) (transform.TransformFunc, error) {
	// Construct the hostpath volume source RNode.
	hostPath, err := createHostPathVolumeSource(path, pathType)
	if err != nil {
		return nil, err
	}
	return SetDaemonSetVolumeFunc(volume, volSrcHostPath, hostPath), nil
}

// SetDaemonSetConfigMapVolumeFunc sets a volume in a DaemonSet for a configmap
// volume source.
func SetDaemonSetConfigMapVolumeFunc(volume string, configmapName string, keyToPaths []corev1.KeyToPath) (transform.TransformFunc, error) {
	// Construct the configmap volume source RNode.
	configMap, err := createConfigMapVolumeSource(configmapName, keyToPaths)
	if err != nil {
		return nil, err
	}
	return SetDaemonSetVolumeFunc(volume, volSrcConfigMap, configMap), nil
}

// SetDaemonSetSecretVolumeFunc sets a volume in a DaemonSet for a secret
// volume source.
func SetDaemonSetSecretVolumeFunc(volume string, secretName string, keyToPaths []corev1.KeyToPath) (transform.TransformFunc, error) {
	// Construct the secret volume source.
	secret, err := createSecretVolumeSource(secretName, keyToPaths)
	if err != nil {
		return nil, err
	}
	return SetDaemonSetVolumeFunc(volume, volSrcSecret, secret), nil
}

// SetDaemonSetVolumeMountFunc sets a volumeMount for a given container in a
// DaemonSet.
func SetDaemonSetVolumeMountFunc(container, volName, mountPath string, mountPropagation corev1.MountPropagationMode) transform.TransformFunc {
	// Create selectors and path.
	containerSelector := fmt.Sprintf("[name=%s]", container)
	volumeMountSelector := fmt.Sprintf("[name=%s]", volName)
	path := []string{"spec", "template", "spec", "containers", containerSelector, "volumeMounts", volumeMountSelector}

	mountPathRNode := kyaml.NewScalarRNode(mountPath)

	var mountPropagationRNode *kyaml.RNode = nil
	if mountPropagation != "" {
		mountPropagationRNode = kyaml.NewScalarRNode(string(mountPropagation))
	}

	return SetVolumeMountFunc(mountPathRNode, mountPropagationRNode, path...)
}

// SetDaemonSetContainerResourceFunc sets the resource requirements of a
// container in a DaemonSet.
func SetDaemonSetContainerResourceFunc(container string, resReq corev1.ResourceRequirements) transform.TransformFunc {
	return func(obj *kyaml.RNode) error {
		// Create selector and path.
		containerSelector := fmt.Sprintf("[name=%s]", container)
		path := []string{"spec", "template", "spec", "containers", containerSelector}

		res, err := goToRNode(resReq)
		if err != nil {
			return err
		}

		tf := SetScalarNodeFunc(resourcesField, res, path...)
		return tf(obj)
	}
}

// SetDaemonSetTolerationFunc sets the pod tolerations in a DaemonSet.
func SetDaemonSetTolerationFunc(tolerations []corev1.Toleration) transform.TransformFunc {
	return func(obj *kyaml.RNode) error {
		// Validate the tolerations.
		for _, toleration := range tolerations {
			if toleration.Operator == corev1.TolerationOpExists && toleration.Value != "" {
				return fmt.Errorf("key(%s): toleration value must be empty when `operator` is 'Exists'", toleration.Key)
			}
		}

		// Construct path and RNode.
		path := []string{"spec", "template", "spec"}
		tols, err := goToRNode(tolerations)
		if err != nil {
			return err
		}

		tf := SetScalarNodeFunc(tolerationsField, tols, path...)
		return tf(obj)
	}
}

// SetDaemonSetNodeSelectorTermsFunc sets the node selector terms for node
// affinity in a DaemonSet.
func SetDaemonSetNodeSelectorTermsFunc(nodeSelectors []corev1.NodeSelectorTerm) transform.TransformFunc {
	return func(obj *kyaml.RNode) error {
		// Create a node selector from the node selector terms.
		// NOTE: Since requiredDuringSchedulingIgnoredDuringExecution accepts a
		// pointer to a NodeSelector, creating a NodeSelector with the given
		// selector terms and assigning the NodeSelector term works better over
		// trying to set field "nodeSelector" in
		// requiredDuringSchedulingIgnoredDuringExecution.
		nodeSelector := corev1.NodeSelector{
			NodeSelectorTerms: nodeSelectors,
		}

		// Construct path and RNode.
		path := []string{"spec", "template", "spec", "affinity", "nodeAffinity"}
		selector, err := goToRNode(nodeSelector)
		if err != nil {
			return err
		}

		return obj.PipeE(
			kyaml.LookupCreate(kyaml.MappingNode, path...),
			kyaml.SetField(nodeSelectorField, selector),
		)
	}
}
