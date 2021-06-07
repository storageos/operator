package transform

import (
	"fmt"

	"github.com/darkowlzz/operator-toolkit/declarative/transform"
	kyaml "sigs.k8s.io/kustomize/kyaml/yaml"
)

// SetClusterRoleBindingSubjectNamespaceFunc sets the namespace of a subject in
// a ClusterRoleBinding.
func SetClusterRoleBindingSubjectNamespaceFunc(name, namespace string) transform.TransformFunc {
	return func(obj *kyaml.RNode) error {
		subjectSelector := fmt.Sprintf("[name=%s]", name)
		return obj.PipeE(
			kyaml.LookupCreate(kyaml.ScalarNode, "subjects", subjectSelector),
			kyaml.SetField("namespace", kyaml.NewScalarRNode(namespace)),
		)
	}
}
