package transform

import (
	"github.com/darkowlzz/operator-toolkit/declarative/transform"
	kyaml "sigs.k8s.io/kustomize/kyaml/yaml"
)

const (
	leaderElection    = "leaderElection"
	resourceNamespace = "resourceNamespace"
)

// SetKubeSchedulerLeaderElectionRNamespaceFunc sets the leader election
// resource namespace in a KubeSchedulerConfiguration.
func SetKubeSchedulerLeaderElectionRNamespaceFunc(namespace string) transform.TransformFunc {
	return func(obj *kyaml.RNode) error {
		// Ensure leaderElection block exists.
		if err := obj.PipeE(kyaml.LookupCreate(kyaml.MappingNode, leaderElection)); err != nil {
			return err
		}
		// Add scalar node.
		return obj.PipeE(
			kyaml.LookupCreate(kyaml.ScalarNode, leaderElection),
			kyaml.SetField(resourceNamespace, kyaml.NewScalarRNode(namespace)),
		)
	}
}
