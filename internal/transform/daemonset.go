package transform

import (
	"fmt"
	"strings"

	"github.com/darkowlzz/operator-toolkit/declarative/transform"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/kustomize/kyaml/yaml"
)

const (
	// Field names of the environment variables.
	envVarValue     = "value"
	envVarValueFrom = "valueFrom"
)

// SetDaemonSetEnvVarFunc sets the environment variable in a DaemonSet
// container for the given key and value field.
func SetDaemonSetEnvVarFunc(container string, key string, valField string, value *yaml.RNode) transform.TransformFunc {
	return func(obj *yaml.RNode) error {
		containerSelector := fmt.Sprintf("[name=%s]", container)
		envVarSelector := fmt.Sprintf("[name=%s]", key)
		return obj.PipeE(
			yaml.LookupCreate(yaml.ScalarNode, "spec", "template", "spec", "containers", containerSelector, "env", envVarSelector),
			yaml.SetField(valField, value),
		)
	}
}

// SetDaemonSetEnvVarStringFunc sets a string value environment variable for a
// given container in a DaemonSet.
func SetDaemonSetEnvVarStringFunc(container, key, val string) transform.TransformFunc {
	return SetDaemonSetEnvVarFunc(container, key, envVarValue, yaml.NewScalarRNode(val))
}

// SetDaemonSetEnvVarValueFromSecretFunc sets a valueFrom secretKeyRef
// environment variable for a given container in a DaemonSet.
func SetDaemonSetEnvVarValueFromSecretFunc(container, key, secretName, secretKey string) (transform.TransformFunc, error) {
	secretKeyRefString := fmt.Sprintf(`
secretKeyRef:
  name: %s
  key: %s
`, secretName, secretKey)
	secretKeyRef, err := yaml.Parse(secretKeyRefString)
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
	fieldRef, err := yaml.Parse(fieldRefString)
	if err != nil {
		return nil, err
	}
	return SetDaemonSetEnvVarFunc(container, key, envVarValueFrom, fieldRef), nil
}

// SetDaemonSetVolumeFunc sets a volume in a DaemonSet for the given name and
// volume source.
func SetDaemonSetVolumeFunc(volume string, volumeSource string, value *yaml.RNode) transform.TransformFunc {
	return func(obj *yaml.RNode) error {
		volumeSelector := fmt.Sprintf("[name=%s]", volume)
		return obj.PipeE(
			yaml.LookupCreate(yaml.ScalarNode, "spec", "template", "spec", "volumes", volumeSelector),
			yaml.SetField(volumeSource, value),
		)
	}
}

// SetDaemonSetHostPathVolumeFunc sets a volume in a DaemonSet for a host path
// volume source.
func SetDaemonSetHostPathVolumeFunc(volume, path, pathType string) (transform.TransformFunc, error) {
	hostPathString := fmt.Sprintf(`
path: %s
type: %s
`, path, pathType)
	hostPath, err := yaml.Parse(hostPathString)
	if err != nil {
		return nil, err
	}
	return SetDaemonSetVolumeFunc(volume, "hostPath", hostPath), nil
}

// constructVolumeSource creates a volume source.
// nameRef is the name of the source type, nameVal is the name of the source.
// It returns a kyaml RNode which can be used in further transformation.
func constructVolumeSource(nameRef string, nameVal string, keyToPaths []corev1.KeyToPath) (*yaml.RNode, error) {
	// Construct the volume source.
	volSrcString := fmt.Sprintf(`%s: %s`, nameRef, nameVal)
	volSrc, err := yaml.Parse(volSrcString)
	if err != nil {
		return nil, err
	}

	// Add items if KeyToPath are provided.
	if len(keyToPaths) > 0 {
		// Construct list of key path items.
		itemsStrings := []string{}
		for _, keyToPath := range keyToPaths {
			itemsString := fmt.Sprintf(`
- key: %s
  path: %s
`, keyToPath.Key, keyToPath.Path)
			itemsStrings = append(itemsStrings, itemsString)
		}

		// Join the key path items to form one list.
		itemsListString := strings.Join(itemsStrings, "\n")

		// Parse the list into a RNode.
		itemsList, err := yaml.Parse(itemsListString)
		if err != nil {
			return nil, err
		}

		// Add keyToPath in the volume source items.
		err = volSrc.PipeE(
			yaml.LookupCreate(yaml.ScalarNode),
			yaml.SetField("items", itemsList),
		)
		if err != nil {
			return nil, err
		}
	}

	return volSrc, nil
}

// SetDaemonSetConfigMapVolumeFunc sets a volume in a DaemonSet for a configmap
// volume source.
func SetDaemonSetConfigMapVolumeFunc(volume string, configmapName string, keyToPaths []corev1.KeyToPath) (transform.TransformFunc, error) {
	// Construct the configmap volume source.
	configMap, err := constructVolumeSource("name", configmapName, keyToPaths)
	if err != nil {
		return nil, err
	}

	// Add the configmap in the volume as configMap volume source.
	return SetDaemonSetVolumeFunc(volume, "configMap", configMap), nil
}

// SetDaemonSetSecretVolumeFunc sets a volume in a DaemonSet for a secret
// volume source.
func SetDaemonSetSecretVolumeFunc(volume string, secretName string, keyToPaths []corev1.KeyToPath) (transform.TransformFunc, error) {
	// Construct the secret volume source.
	secret, err := constructVolumeSource("secretName", secretName, keyToPaths)
	if err != nil {
		return nil, err
	}

	// Add the configmap in the volume as configMap volume source.
	return SetDaemonSetVolumeFunc(volume, "secret", secret), nil
}
