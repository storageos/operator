package transform

import (
	"fmt"
	"strconv"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	kyaml "sigs.k8s.io/kustomize/kyaml/yaml"
)

func TestSetDefaultServicePortNameFunc(t *testing.T) {
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
  type: ClusterIP
`)
	assert.Nil(t, err)

	wantName := "foo-svc"

	// Transform.
	tf := SetDefaultServicePortNameFunc(wantName)
	err = tf(testObj)
	assert.Nil(t, err)

	// Query and check the value.
	portSelector := fmt.Sprintf("[name=%s]", wantName)
	portName, err := testObj.Pipe(kyaml.Lookup("spec", "ports", portSelector, "name"))
	assert.Nil(t, err)
	str, err := portName.String()
	assert.Nil(t, err)
	assert.Equal(t, wantName, strings.TrimSpace(str))
}

func TestSetServiceTypeFunc(t *testing.T) {
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
  type: ClusterIP
`)
	assert.Nil(t, err)

	wantType := corev1.ServiceTypeNodePort

	// Transform.
	tf := SetServiceTypeFunc(wantType)
	err = tf(testObj)
	assert.Nil(t, err)

	// Query and check the value.
	typeName, err := testObj.Pipe(kyaml.Lookup("spec", "type"))
	assert.Nil(t, err)
	str, err := typeName.String()
	assert.Nil(t, err)
	assert.Equal(t, string(wantType), strings.TrimSpace(str))
}

func TestSetServiceInternalPortFunc(t *testing.T) {
	testObj, err := kyaml.Parse(`
apiVersion: v1
kind: Service
metadata:
  name: appsvc
spec:
  ports:
  - name: svc1
    port: 7777
    protocol: TCP
    targetPort: 9999
  - name: svc2
    port: 2222
    protocol: TCP
    targetPort: 6666
  type: ClusterIP
`)
	assert.Nil(t, err)

	cases := []struct {
		name     string
		portName string
		port     int
	}{
		{
			name:     "set first port",
			portName: "svc1",
			port:     1111,
		},
		{
			name:     "set second port",
			portName: "svc2",
			port:     5555,
		},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			// Make a copy of the object.
			obj := testObj.Copy()

			// Transform.
			tf := SetServiceInternalPortFunc(tc.portName, tc.port)
			err = tf(obj)
			assert.Nil(t, err)

			// Query and check the value.
			portSelector := fmt.Sprintf("[name=%s]", tc.portName)
			port, err := obj.Pipe(kyaml.Lookup("spec", "ports", portSelector, "port"))
			assert.Nil(t, err)
			str, err := port.String()
			assert.Nil(t, err)
			assert.Equal(t, strconv.Itoa(tc.port), strings.TrimSpace(str))
		})
	}
}

func TestSetServiceExternalPortFunc(t *testing.T) {
	testObj, err := kyaml.Parse(`
apiVersion: v1
kind: Service
metadata:
  name: appsvc
spec:
  ports:
  - name: svc1
    port: 7777
    protocol: TCP
    targetPort: 9999
  - name: svc2
    port: 2222
    protocol: TCP
    targetPort: 6666
  type: ClusterIP
`)
	assert.Nil(t, err)

	cases := []struct {
		name     string
		portName string
		port     int
	}{
		{
			name:     "set first port",
			portName: "svc1",
			port:     1111,
		},
		{
			name:     "set second port",
			portName: "svc2",
			port:     5555,
		},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			// Make a copy of the object.
			obj := testObj.Copy()

			// Transform.
			tf := SetServiceExternalPortFunc(tc.portName, tc.port)
			err = tf(obj)
			assert.Nil(t, err)

			// Query and check the value.
			portSelector := fmt.Sprintf("[name=%s]", tc.portName)
			port, err := obj.Pipe(kyaml.Lookup("spec", "ports", portSelector, "targetPort"))
			assert.Nil(t, err)
			str, err := port.String()
			assert.Nil(t, err)
			assert.Equal(t, strconv.Itoa(tc.port), strings.TrimSpace(str))
		})
	}
}
