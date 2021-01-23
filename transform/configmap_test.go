package transform

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"sigs.k8s.io/kustomize/kyaml/yaml"
)

func TestSetConfigMapData(t *testing.T) {
	testObj, err := yaml.Parse(`
apiVersion: v1
kind: ConfigMap
metadata:
  name: some-config
data:
  LOG_FORMAT: json
  DISABLE_X: "true"
`)
	assert.Nil(t, err)

	cases := []struct {
		name  string
		key   string
		value string
	}{
		{
			name:  "add new configmap data",
			key:   "CONFIG1",
			value: "VAL1",
		},
		{
			name:  "overwrite existing env var",
			key:   "LOG_FORMAT",
			value: "raw",
		},
		{
			name:  "add a boolean data",
			key:   "TELEMETRY",
			value: "true",
		},
		{
			name:  "overwrite existing boolean",
			key:   "DISABLE_X",
			value: "false",
		},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			// Make a copy of the object.
			obj := testObj.Copy()

			// Transform.
			tf := SetConfigMapData(tc.key, tc.value)
			err = tf(obj)
			assert.Nil(t, err)

			// Query and check the value.
			val, err := obj.Pipe(yaml.Lookup("data", tc.key))
			assert.Nil(t, err)

			str, err := val.String()
			assert.Nil(t, err)
			// Trim newline and the quotes around the value.
			str = strings.TrimSpace(str)
			assert.Equal(t, tc.value, strings.Trim(str, "\""))
		})
	}
}
