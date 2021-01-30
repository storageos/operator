package transform

import (
	"fmt"

	"github.com/darkowlzz/operator-toolkit/declarative/transform"
	corev1 "k8s.io/api/core/v1"
	kyaml "sigs.k8s.io/kustomize/kyaml/yaml"
	"sigs.k8s.io/yaml"
)

const (
	// Field names of the environment variables.
	envVarValue     = "value"
	envVarValueFrom = "valueFrom"

	// Field names in volume mounts.
	volMountPath        = "mountPath"
	volMountPropagation = "mountPropagation"
)

// SetDaemonSetEnvVarFunc sets the environment variable in a DaemonSet
// container for the given key and value field.
func SetDaemonSetEnvVarFunc(container string, key string, valField string, value *kyaml.RNode) transform.TransformFunc {
	return func(obj *kyaml.RNode) error {
		containerSelector := fmt.Sprintf("[name=%s]", container)
		envVarSelector := fmt.Sprintf("[name=%s]", key)
		return obj.PipeE(
			kyaml.LookupCreate(kyaml.ScalarNode, "spec", "template", "spec", "containers", containerSelector, "env", envVarSelector),
			kyaml.SetField(valField, value),
		)
	}
}

// SetDaemonSetEnvVarStringFunc sets a string value environment variable for a
// given container in a DaemonSet.
func SetDaemonSetEnvVarStringFunc(container, key, val string) transform.TransformFunc {
	return SetDaemonSetEnvVarFunc(container, key, envVarValue, kyaml.NewScalarRNode(val))
}

// SetDaemonSetEnvVarValueFromSecretFunc sets a valueFrom secretKeyRef
// environment variable for a given container in a DaemonSet.
func SetDaemonSetEnvVarValueFromSecretFunc(container, key, secretName, secretKey string) (transform.TransformFunc, error) {
	secretKeyRefString := fmt.Sprintf(`
secretKeyRef:
  name: %s
  key: %s
`, secretName, secretKey)
	secretKeyRef, err := kyaml.Parse(secretKeyRefString)
	if err != nil {
		return nil, err
	}
	return SetDaemonSetEnvVarFunc(container, key, envVarValueFrom, secretKeyRef), nil
}

// SetDaemonSetEnvVarValueFromFieldFunc sets a valueFrom fieldRef environment
// variable for a given container in a DaemonSet.
func SetDaemonSetEnvVarValueFromFieldFunc(container, key, fieldPath string) (transform.TransformFunc, error) {
	fieldRefString := fmt.Sprintf(`
fieldRef:
  apiVersion: v1
  fieldPath: %s
`, fieldPath)
	fieldRef, err := kyaml.Parse(fieldRefString)
	if err != nil {
		return nil, err
	}
	return SetDaemonSetEnvVarFunc(container, key, envVarValueFrom, fieldRef), nil
}

// SetDaemonSetVolumeFunc sets a volume in a DaemonSet for the given name and
// volume source.
func SetDaemonSetVolumeFunc(volume string, volumeSource string, value *kyaml.RNode) transform.TransformFunc {
	return func(obj *kyaml.RNode) error {
		volumeSelector := fmt.Sprintf("[name=%s]", volume)
		return obj.PipeE(
			kyaml.LookupCreate(kyaml.ScalarNode, "spec", "template", "spec", "volumes", volumeSelector),
			kyaml.SetField(volumeSource, value),
		)
	}
}

// SetDaemonSetHostPathVolumeFunc sets a volume in a DaemonSet for a host path
// volume source.
func SetDaemonSetHostPathVolumeFunc(volume, path, pathType string) (transform.TransformFunc, error) {
	// Construct the hostpath volume source.
	hostPath, err := createHostPathVolumeSource(path, pathType)
	if err != nil {
		return nil, err
	}

	// Return a transform func to set the hostpath volume source.
	return SetDaemonSetVolumeFunc(volume, volSrcHostPath, hostPath), nil
}

// SetDaemonSetConfigMapVolumeFunc sets a volume in a DaemonSet for a configmap
// volume source.
func SetDaemonSetConfigMapVolumeFunc(volume string, configmapName string, keyToPaths []corev1.KeyToPath) (transform.TransformFunc, error) {
	// Construct the configmap volume source.
	configMap, err := createKeyValVolumeSource("name", configmapName, keyToPaths)
	if err != nil {
		return nil, err
	}

	// Return a transform func to set the configmap volume source.
	return SetDaemonSetVolumeFunc(volume, volSrcConfigMap, configMap), nil
}

// SetDaemonSetSecretVolumeFunc sets a volume in a DaemonSet for a secret
// volume source.
func SetDaemonSetSecretVolumeFunc(volume string, secretName string, keyToPaths []corev1.KeyToPath) (transform.TransformFunc, error) {
	// Construct the secret volume source.
	secret, err := createKeyValVolumeSource("secretName", secretName, keyToPaths)
	if err != nil {
		return nil, err
	}

	// Return a transform func to set the secret volume source.
	return SetDaemonSetVolumeFunc(volume, volSrcSecret, secret), nil
}

// SetDaemonSetVolumeMountFunc sets a volumeMount for a given container in a
// DaemonSet.
func SetDaemonSetVolumeMountFunc(container, volName, mountPath string, mountPropagation corev1.MountPropagationMode) transform.TransformFunc {
	return func(obj *kyaml.RNode) error {
		// Create selectors.
		containerSelector := fmt.Sprintf("[name=%s]", container)
		volumeMountSelector := fmt.Sprintf("[name=%s]", volName)

		// Add mount path.
		err := obj.PipeE(
			kyaml.LookupCreate(kyaml.ScalarNode, "spec", "template", "spec", "containers", containerSelector, "volumeMounts", volumeMountSelector),
			kyaml.SetField(volMountPath, kyaml.NewScalarRNode(mountPath)),
		)
		if err != nil {
			return err
		}

		// Add mount propagation if provided.
		if mountPropagation != "" {
			err := obj.PipeE(
				kyaml.LookupCreate(kyaml.ScalarNode, "spec", "template", "spec", "containers", containerSelector, "volumeMounts", volumeMountSelector),
				kyaml.SetField(volMountPropagation, kyaml.NewScalarRNode(string(mountPropagation))),
			)
			if err != nil {
				return err
			}
		}

		return nil
	}
}

// SetDaemonSetContainerResourceFunc sets the resource requirements of a
// container in a DaemonSet.
func SetDaemonSetContainerResourceFunc(container string, resReq corev1.ResourceRequirements) transform.TransformFunc {
	return func(obj *kyaml.RNode) error {
		// Create selectors.
		containerSelector := fmt.Sprintf("[name=%s]", container)

		// Convert go typed resource to yaml.
		// NOTE: Using sigs.k8s.io/yaml for correct decoding of the values.
		// Using kyaml's Marshal doesn't decode with proper values.
		//
		// Following is an example result of using kyaml marshal:
		//   limits:
		//	   cpu:
		//		   format: DecimalSI
		//	   memory:
		//		   format: BinarySI
		//
		// Following is an example result of using sigs.k8s.io/yaml marshal:
		//   limits:
		//     cpu: 200m
		//     memory: 500Mi
		resReqBytes, err := yaml.Marshal(resReq)
		if err != nil {
			return err
		}

		// Parse the yaml resource to create an RNode.
		resources, err := kyaml.Parse(string(resReqBytes))
		if err != nil {
			return err
		}

		return obj.PipeE(
			kyaml.LookupCreate(kyaml.ScalarNode, "spec", "template", "spec", "containers", containerSelector),
			kyaml.SetField("resources", resources),
		)
	}
}
