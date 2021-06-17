package transform

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	kyaml "sigs.k8s.io/kustomize/kyaml/yaml"
)

func TestSetKubeSchedulerLeaderElectionRNamespaceFunc(t *testing.T) {
	testObj, err := kyaml.Parse(`
apiVersion: kubescheduler.config.k8s.io/v1beta1
kind: KubeSchedulerConfiguration
profiles:
  - schedulerName: foo-scheduler
`)
	assert.Nil(t, err)

	testObjWithLeaderElect, err := kyaml.Parse(`
apiVersion: kubescheduler.config.k8s.io/v1beta1
kind: KubeSchedulerConfiguration
profiles:
  - schedulerName: foo-scheduler
leaderElection:
  leaderElect: true
`)
	assert.Nil(t, err)

	testObjWithResourceNS, err := kyaml.Parse(`
apiVersion: kubescheduler.config.k8s.io/v1beta1
kind: KubeSchedulerConfiguration
profiles:
  - schedulerName: foo-scheduler
leaderElection:
  leaderElect: true
  resourceNamespace: some-ns
`)
	assert.Nil(t, err)

	cases := []struct {
		name                string
		obj                 *kyaml.RNode
		resourceNamespace   string
		wantSchedulerConfig string
	}{
		{
			name:              "no leaderElection set",
			obj:               testObj,
			resourceNamespace: "namespaceA",
			wantSchedulerConfig: `
apiVersion: kubescheduler.config.k8s.io/v1beta1
kind: KubeSchedulerConfiguration
profiles:
  - schedulerName: foo-scheduler
leaderElection:
  resourceNamespace: namespaceA
`,
		},
		{
			name:              "leaderElection set",
			obj:               testObjWithLeaderElect,
			resourceNamespace: "namespaceB",
			wantSchedulerConfig: `
apiVersion: kubescheduler.config.k8s.io/v1beta1
kind: KubeSchedulerConfiguration
profiles:
  - schedulerName: foo-scheduler
leaderElection:
  leaderElect: true
  resourceNamespace: namespaceB
`,
		},
		{
			name:              "resourceNamespace set",
			obj:               testObjWithResourceNS,
			resourceNamespace: "namespaceC",
			wantSchedulerConfig: `
apiVersion: kubescheduler.config.k8s.io/v1beta1
kind: KubeSchedulerConfiguration
profiles:
  - schedulerName: foo-scheduler
leaderElection:
  leaderElect: true
  resourceNamespace: namespaceC
`,
		},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			obj := tc.obj.Copy()

			tf := SetKubeSchedulerLeaderElectionRNamespaceFunc(tc.resourceNamespace)
			err = tf(obj)
			assert.Nil(t, err)

			// Check the result.
			gotStr, err := obj.String()
			assert.Nil(t, err)
			assert.Equal(t, strings.TrimSpace(tc.wantSchedulerConfig), strings.TrimSpace(gotStr))
		})
	}
}
