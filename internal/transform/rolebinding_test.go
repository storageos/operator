package transform

import (
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	kyaml "sigs.k8s.io/kustomize/kyaml/yaml"
)

func TestSetClusterRoleBindingSubjectNamespaceFunc(t *testing.T) {
	testObj, err := kyaml.Parse(`
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRoleBinding
metadata:
  name: app-role-binding
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: ClusterRole
  name: app-role
subjects:
- kind: ServiceAccount
  name: app-sa-1
  namespace: default
- kind: ServiceAccount
  name: app-sa-2
  namespace: default
`)
	assert.Nil(t, err)

	cases := []struct {
		name        string
		subjectName string
		namespace   string
	}{
		{
			name:        "set first subject",
			subjectName: "app-sa-1",
			namespace:   "foo-ns",
		},
		{
			name:        "set second subject",
			subjectName: "app-sa-2",
			namespace:   "bar-ns",
		},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			// Make a copy of the object.
			obj := testObj.Copy()

			// Transform.
			tf := SetClusterRoleBindingSubjectNamespaceFunc(tc.subjectName, tc.namespace)
			err = tf(obj)
			assert.Nil(t, err)

			// Query and check the value.
			subjectSelector := fmt.Sprintf("[name=%s]", tc.subjectName)
			namespace, err := obj.Pipe(kyaml.Lookup("subjects", subjectSelector, "namespace"))
			assert.Nil(t, err)
			str, err := namespace.String()
			assert.Nil(t, err)
			assert.Equal(t, tc.namespace, strings.TrimSpace(str))
		})
	}
}
