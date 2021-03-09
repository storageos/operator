package transform

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	kyaml "sigs.k8s.io/kustomize/kyaml/yaml"
)

func TestSetMetadataNameFunc(t *testing.T) {
	testObj, err := kyaml.Parse(`
apiVersion: v1
kind: Service
metadata:
  name: storageos
spec:
  ports:
  - name: storageos
    port: 5705
    protocol: TCP
    targetPort: 5705
  sessionAffinity: None
  type: ClusterIP
`)
	assert.Nil(t, err)

	wantName := "foo-svc"

	// Transform.
	tf := SetMetadataNameFunc(wantName)
	err = tf(testObj)
	assert.Nil(t, err)

	// Query and check the value.
	val, err := testObj.Pipe(kyaml.Lookup("metadata", "name"))
	assert.Nil(t, err)
	str, err := val.String()
	assert.Nil(t, err)
	assert.Equal(t, wantName, strings.TrimSpace(str))
}
