package transform

import (
	"github.com/darkowlzz/operator-toolkit/declarative/transform"
	kyaml "sigs.k8s.io/kustomize/kyaml/yaml"
)

// SetEnvVarStringFunc sets a string value environment variable for a given
// container path.
func SetEnvVarStringFunc(val string, path ...string) transform.TransformFunc {
	return SetScalarNodeFunc(envVarValue, kyaml.NewScalarRNode(val), path...)
}

// SetEnvVarValueFromSecretFunc sets a valueFrom secretKeyRef environment
// variable for a given container path.
func SetEnvVarValueFromSecretFunc(secretName, secretKey string, path ...string) transform.TransformFunc {
	return func(obj *kyaml.RNode) error {
		// Construct secret env var source RNode.
		envVarSrc, err := createSecretEnvVarSource(secretName, secretKey)
		if err != nil {
			return err
		}
		tf := SetScalarNodeFunc(envVarValueFrom, envVarSrc, path...)
		return tf(obj)
	}
}

// SetEnvVarValueFromFieldFunc sets a valueFrom fieldRef environment variable
// for a given container path.
func SetEnvVarValueFromFieldFunc(fieldPath string, path ...string) transform.TransformFunc {
	return func(obj *kyaml.RNode) error {
		// Construct env var source from field ref RNode.
		envVarSrc, err := createValueFromFieldEnvVarSource(fieldPath)
		if err != nil {
			return err
		}
		tf := SetScalarNodeFunc(envVarValueFrom, envVarSrc, path...)
		return tf(obj)
	}
}
