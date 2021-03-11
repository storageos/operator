package transform

import (
	"fmt"

	"github.com/darkowlzz/operator-toolkit/declarative/transform"
	kyaml "sigs.k8s.io/kustomize/kyaml/yaml"
)

// getDeploymentContainerArgsPath constructs path to container args in a Deployment.
func getDeploymentContainerArgsPath(containerType, container string) []string {
	containerSelector := fmt.Sprintf("[name=%s]", container)
	return []string{"spec", "template", "spec", containerType, containerSelector, "args"}
}

// AppendDeploymentContainerArgsFunc adds a list of args in a given container
// in a Deployment.
func AppendDeploymentContainerArgsFunc(container string, vals []string) transform.TransformFunc {
	path := getDeploymentContainerArgsPath(containerTypeMain, container)
	return AppendSequenceNodeFunc(kyaml.NewListRNode(vals...), path...)
}
