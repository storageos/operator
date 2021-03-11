package transform

import (
	"fmt"

	"github.com/darkowlzz/operator-toolkit/declarative/transform"
	corev1 "k8s.io/api/core/v1"
	kyaml "sigs.k8s.io/kustomize/kyaml/yaml"
)

// getPodTemplateContainerArgsPath constructs path to container args in a PodTemplate.
func getPodTemplateContainerArgsPath(containerType, container string) []string {
	containerSelector := fmt.Sprintf("[name=%s]", container)
	return []string{"spec", "template", "spec", containerType, containerSelector, "args"}
}

// AppendPodTemplateContainerArgsFunc adds a list of args in a given container
// in a PodTemplate.
func AppendPodTemplateContainerArgsFunc(container string, vals []string) transform.TransformFunc {
	path := getPodTemplateContainerArgsPath(containerTypeMain, container)
	return AppendSequenceNodeFunc(kyaml.NewListRNode(vals...), path...)
}

// getPodTemplateEnvVarPath constructs path to an env var in a PodTemplate.
func getPodTemplateEnvVarPath(containerType, container, key string) []string {
	containerSelector := fmt.Sprintf("[name=%s]", container)
	envVarSelector := fmt.Sprintf("[name=%s]", key)
	return []string{"spec", "template", "spec", containerType, containerSelector, "env", envVarSelector}
}

// SetPodTemplateContainerEnvVarStringFunc sets a string value env var for a
// container.
func SetPodTemplateContainerEnvVarStringFunc(container, key, val string) transform.TransformFunc {
	path := getPodTemplateEnvVarPath(containerTypeMain, container, key)
	return SetEnvVarStringFunc(val, path...)
}

// SetPodTemplateContainerEnvVarValueFromSecretFunc sets a valueFrom secretKeyRef
// env var for a container.
func SetPodTemplateContainerEnvVarValueFromSecretFunc(container, key, secretName, secretKey string) transform.TransformFunc {
	path := getPodTemplateEnvVarPath(containerTypeMain, container, key)
	return SetEnvVarValueFromSecretFunc(secretName, secretKey, path...)
}

// SetPodTemplateContainerEnvVarValueFromFieldFunc sets a valueFrom fieldRef env
// var for a container.
func SetPodTemplateContainerEnvVarValueFromFieldFunc(container, key, fieldPath string) transform.TransformFunc {
	path := getPodTemplateEnvVarPath(containerTypeMain, container, key)
	return SetEnvVarValueFromFieldFunc(fieldPath, path...)
}

// SetPodTemplateInitContainerEnvVarStringFunc sets a string value env var for an
// init container.
func SetPodTemplateInitContainerEnvVarStringFunc(container, key, val string) transform.TransformFunc {
	path := getPodTemplateEnvVarPath(containerTypeInit, container, key)
	return SetEnvVarStringFunc(val, path...)
}

// SetPodTemplateInitContainerEnvVarValueFromSecretFunc sets a valueFrom
// secretKeyRef env var for an init container.
func SetPodTemplateInitContainerEnvVarValueFromSecretFunc(container, key, secretName, secretKey string) transform.TransformFunc {
	path := getPodTemplateEnvVarPath(containerTypeInit, container, key)
	return SetEnvVarValueFromSecretFunc(secretName, secretKey, path...)
}

// SetPodTemplateInitContainerEnvVarValueFromFieldFunc sets a valueFrom fieldRef
// env var for an init container.
func SetPodTemplateInitContainerEnvVarValueFromFieldFunc(container, key, fieldPath string) transform.TransformFunc {
	path := getPodTemplateEnvVarPath(containerTypeInit, container, key)
	return SetEnvVarValueFromFieldFunc(fieldPath, path...)
}

// SetPodTemplateVolumeFunc sets a volume in a PodTemplate for the given name
// and volume source.
func SetPodTemplateVolumeFunc(volume string, volumeSource string, value *kyaml.RNode) transform.TransformFunc {
	volumeSelector := fmt.Sprintf("[name=%s]", volume)
	path := []string{"spec", "template", "spec", "volumes", volumeSelector}
	return SetScalarNodeFunc(volumeSource, value, path...)
}

// SetPodTemplateHostPathVolumeFunc sets a volume in a PodTemplate for a host path
// volume source.
func SetPodTemplateHostPathVolumeFunc(volume, path string, pathType *corev1.HostPathType) transform.TransformFunc {
	return func(obj *kyaml.RNode) error {
		// Construct the hostpath volume source RNode.
		hostPath, err := createHostPathVolumeSource(path, pathType)
		if err != nil {
			return err
		}
		tf := SetPodTemplateVolumeFunc(volume, volSrcHostPath, hostPath)
		return tf(obj)
	}
}

// SetPodTemplateConfigMapVolumeFunc sets a volume in a PodTemplate for a configmap
// volume source.
func SetPodTemplateConfigMapVolumeFunc(volume string, configmapName string, keyToPaths []corev1.KeyToPath) transform.TransformFunc {
	return func(obj *kyaml.RNode) error {
		// Construct the configmap volume source RNode.
		configMap, err := createConfigMapVolumeSource(configmapName, keyToPaths)
		if err != nil {
			return err
		}
		tf := SetPodTemplateVolumeFunc(volume, volSrcConfigMap, configMap)
		return tf(obj)
	}
}

// SetPodTemplateSecretVolumeFunc sets a volume in a PodTemplate for a secret
// volume source.
func SetPodTemplateSecretVolumeFunc(volume string, secretName string, keyToPaths []corev1.KeyToPath) transform.TransformFunc {
	return func(obj *kyaml.RNode) error {
		// Construct the secret volume source.
		secret, err := createSecretVolumeSource(secretName, keyToPaths)
		if err != nil {
			return err
		}
		tf := SetPodTemplateVolumeFunc(volume, volSrcSecret, secret)
		return tf(obj)
	}
}

// SetPodTemplateVolumeMountFunc sets a volumeMount for a given container in a
// PodTemplate.
func SetPodTemplateVolumeMountFunc(container, volName, mountPath string, mountPropagation corev1.MountPropagationMode) transform.TransformFunc {
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

// SetPodTemplateContainerResourceFunc sets the resource requirements of a
// container in a PodTemplate.
func SetPodTemplateContainerResourceFunc(container string, resReq corev1.ResourceRequirements) transform.TransformFunc {
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

// SetPodTemplateTolerationFunc sets the pod tolerations in a PodTemplate.
func SetPodTemplateTolerationFunc(tolerations []corev1.Toleration) transform.TransformFunc {
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

// SetPodTemplateNodeSelectorTermsFunc sets the node selector terms for node
// affinity in a PodTemplate.
func SetPodTemplateNodeSelectorTermsFunc(nodeSelectors []corev1.NodeSelectorTerm) transform.TransformFunc {
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
