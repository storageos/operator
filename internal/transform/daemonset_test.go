package transform

import (
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"sigs.k8s.io/kustomize/kyaml/yaml"
)

func TestSetDaemonSetEnvVarStringFunc(t *testing.T) {
	testObj, err := yaml.Parse(`
apiVersion: apps/v1
kind: DaemonSet
metadata:
  name: some-daemonset
spec:
  template:
    spec:
      containers:
      - name: myapp
        image: some-app:v1.1.1
        args:
        - server
      - name: someotherapp
        image: some-image:v1.2.3
        args:
        - --v=5
        env:
        - name: ADDRESS
          value: /xyz/abc.sock
`)
	assert.Nil(t, err)

	cases := []struct {
		name      string
		container string
		key       string
		val       string
	}{
		{
			name:      "add new env var",
			container: "myapp",
			key:       "NEW_ENV_VAR",
			val:       "some-new-value",
		},
		{
			name:      "overwrite existing env var",
			container: "someotherapp",
			key:       "ADDRESS",
			val:       "some-address.sock",
		},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			// Make a copy of the object.
			obj := testObj.Copy()

			// Transform.
			tf := SetDaemonSetEnvVarStringFunc(tc.container, tc.key, tc.val)
			err = tf(obj)
			assert.Nil(t, err)

			// Query and check the value.
			containerSelector := fmt.Sprintf("[name=%s]", tc.container)
			envVarSelector := fmt.Sprintf("[name=%s]", tc.key)
			val, err := obj.Pipe(yaml.Lookup("spec", "template", "spec", "containers", containerSelector, "env", envVarSelector, "value"))
			assert.Nil(t, err)

			str, err := val.String()
			assert.Nil(t, err)
			assert.Equal(t, tc.val, strings.TrimSpace(str))
		})
	}
}

func TestSetDaemonSetEnvVarValueFromSecretFunc(t *testing.T) {
	testObj, err := yaml.Parse(`
apiVersion: apps/v1
kind: DaemonSet
metadata:
  name: some-daemonset
spec:
  template:
    spec:
      containers:
      - name: someotherapp
        image: some-image:v1.2.3
        env:
        - name: ADDRESS
          value: /xyz/abc.sock
        - name: SECRET_USER
          valueFrom:
            secretKeyRef:
              key: username
              name: init-secret
`)
	assert.Nil(t, err)

	cases := []struct {
		name       string
		container  string
		key        string
		secretName string
		secretKey  string
		wantVal    string
	}{
		{
			name:       "new value from secret",
			container:  "someotherapp",
			key:        "SOME_VAR_FROM_SECRET",
			secretName: "init-secret",
			secretKey:  "username",
			wantVal: `
secretKeyRef:
  name: init-secret
  key: username`,
		},
		{
			name:       "overwrite existing env var",
			container:  "someotherapp",
			key:        "SECRET_USER",
			secretName: "creds",
			secretKey:  "user",
			wantVal: `
secretKeyRef:
  name: creds
  key: user`,
		},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			// Make a copy of the object.
			obj := testObj.Copy()
			tf, err := SetDaemonSetEnvVarValueFromSecretFunc(tc.container, tc.key, tc.secretName, tc.secretKey)
			assert.Nil(t, err)
			err = tf(obj)
			assert.Nil(t, err)

			containerSelector := fmt.Sprintf("[name=%s]", tc.container)
			envVarSelector := fmt.Sprintf("[name=%s]", tc.key)
			val, err := obj.Pipe(yaml.Lookup("spec", "template", "spec", "containers", containerSelector, "env", envVarSelector, "valueFrom"))
			assert.Nil(t, err)

			str, err := val.String()
			assert.Nil(t, err)
			assert.Equal(t, strings.TrimSpace(tc.wantVal), strings.TrimSpace(str))
		})
	}
}

func TestSetDaemonSetEnvVarValueFromFieldFunc(t *testing.T) {
	testObj, err := yaml.Parse(`
apiVersion: apps/v1
kind: DaemonSet
metadata:
  name: some-daemonset
spec:
  template:
    spec:
      containers:
      - name: someotherapp
        image: some-image:v1.2.3
        env:
        - name: ADDRESS
          value: /xyz/abc.sock
        - name: POD_IP
          valueFrom:
            fieldRef:
              apiVersion: v1
              fieldPath: status.podIP
`)
	assert.Nil(t, err)

	cases := []struct {
		name      string
		container string
		key       string
		fieldPath string
		wantVal   string
	}{
		{
			name:      "new value from field",
			container: "someotherapp",
			key:       "SOME_VAR_FROM_FIELD",
			fieldPath: "status.running",
			wantVal: `
fieldRef:
  apiVersion: v1
  fieldPath: status.running`,
		},
		{
			name:      "overwrite existing env var",
			container: "someotherapp",
			key:       "POD_IP",
			fieldPath: "status.podIP",
			wantVal: `
fieldRef:
  apiVersion: v1
  fieldPath: status.podIP`,
		},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			// Make a copy of the object.
			obj := testObj.Copy()
			tf, err := SetDaemonSetEnvVarValueFromFieldFunc(tc.container, tc.key, tc.fieldPath)
			assert.Nil(t, err)
			err = tf(obj)
			assert.Nil(t, err)

			containerSelector := fmt.Sprintf("[name=%s]", tc.container)
			envVarSelector := fmt.Sprintf("[name=%s]", tc.key)
			val, err := obj.Pipe(yaml.Lookup("spec", "template", "spec", "containers", containerSelector, "env", envVarSelector, "valueFrom"))
			assert.Nil(t, err)

			str, err := val.String()
			assert.Nil(t, err)
			assert.Equal(t, strings.TrimSpace(tc.wantVal), strings.TrimSpace(str))
		})
	}
}
