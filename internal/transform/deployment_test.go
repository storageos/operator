package transform

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	kyaml "sigs.k8s.io/kustomize/kyaml/yaml"
)

func TestAppendDeploymentContainerArgsFunc(t *testing.T) {
	testObj, err := kyaml.Parse(`
apiVersion: apps/v1
kind: Deployment
metadata:
  name: some-app
spec:
  replicas: 1
  strategy:
    type: RollingUpdate
  template:
    spec:
      containers:
      - args:
        - server
        - --leader-elect=true
        image: some-image:v1.0.0
        name: myapp
      - name: my-other-app
        image: some-other-image:v2.0.0
`)
	assert.Nil(t, err)

	cases := []struct {
		name      string
		container string
		args      []string
		want      []string
	}{
		{
			name:      "add new args",
			container: "my-other-app",
			args:      []string{"some-new-value", "more-new-value"},
			want:      []string{"some-new-value", "more-new-value"},
		},
		{
			name:      "append to existing args",
			container: "myapp",
			args:      []string{"some-new-value", "more-new-value"},
			want:      []string{"server", "--leader-elect=true", "some-new-value", "more-new-value"},
		},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			// Make a copy of the object.
			obj := testObj.Copy()

			// Transform.
			tf := AppendDeploymentContainerArgsFunc(tc.container, tc.args)
			err = tf(obj)
			assert.Nil(t, err)

			// Query and check value.
			containerSelector := fmt.Sprintf("[name=%s]", tc.container)
			val, err := obj.Pipe(kyaml.Lookup("spec", "template", "spec", "containers", containerSelector, "args"))
			assert.Nil(t, err)

			str, err := val.String()
			assert.Nil(t, err)

			// Convert want list to RNode string value.
			wantRNode := kyaml.NewListRNode(tc.want...)
			wantStr, err := wantRNode.String()
			assert.Nil(t, err)

			assert.Equal(t, str, wantStr)
		})
	}
}
