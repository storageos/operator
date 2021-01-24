package transform

import (
	"github.com/darkowlzz/operator-toolkit/declarative/transform"
	"sigs.k8s.io/kustomize/kyaml/yaml"
)

// SetConfigMapData sets the given key-value data in a ConfigMap.
func SetConfigMapData(key, value string) transform.TransformFunc {
	// Ensure the value is double quoted.
	val := yaml.NewScalarRNode(value)
	val.YNode().Style = yaml.DoubleQuotedStyle

	return func(obj *yaml.RNode) error {
		return obj.PipeE(
			yaml.LookupCreate(yaml.ScalarNode, "data"),
			yaml.SetField(key, val),
		)
	}
}
