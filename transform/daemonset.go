package transform

import (
	"fmt"

	"github.com/darkowlzz/operator-toolkit/declarative/transform"
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
