package transform

import (
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
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

func TestSetDaemonSetHostPathVolumeFunc(t *testing.T) {
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
      volumes:
      - name: foo
        hostPath:
          path: /var/lib/
          type: Directory
`)
	assert.Nil(t, err)

	cases := []struct {
		name     string
		volume   string
		path     string
		pathType string
		wantVal  string
	}{
		{
			name:     "new volume",
			volume:   "somedir",
			path:     "/var/lib/foo",
			pathType: "Directory",
			wantVal: `
name: somedir
hostPath:
  path: /var/lib/foo
  type: Directory`,
		},
		{
			name:     "file type",
			volume:   "somefile",
			path:     "/xyz",
			pathType: "File",
			wantVal: `
name: somefile
hostPath:
  path: /xyz
  type: File`,
		},
		{
			name:     "overwrite existing volume",
			volume:   "foo",
			path:     "/usr/local/bin/foo",
			pathType: "File",
			wantVal: `
name: foo
hostPath:
  path: /usr/local/bin/foo
  type: File`,
		},
		{
			name:   "no path type",
			volume: "somedir",
			path:   "/xyz",
			wantVal: `
name: somedir
hostPath:
  path: /xyz
  type:
`,
		},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			obj := testObj.Copy()

			// Transform.
			tf, err := SetDaemonSetHostPathVolumeFunc(tc.volume, tc.path, tc.pathType)
			assert.Nil(t, err)
			err = tf(obj)
			assert.Nil(t, err)

			volumeSelector := fmt.Sprintf("[name=%s]", tc.volume)
			val, err := obj.Pipe(yaml.Lookup("spec", "template", "spec", "volumes", volumeSelector))
			assert.Nil(t, err)

			str, err := val.String()
			assert.Nil(t, err)
			assert.Equal(t, strings.TrimSpace(tc.wantVal), strings.TrimSpace(str))
		})
	}
}

func TestSetDaemonSetConfigMapVolumeFunc(t *testing.T) {
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
      volumes:
      - name: foo
        configMap:
          name: app-config
`)
	assert.Nil(t, err)

	cases := []struct {
		name          string
		volume        string
		configMapName string
		keyToPaths    []corev1.KeyToPath
		wantVal       string
	}{
		{
			name:          "add new volume",
			volume:        "config-vol",
			configMapName: "my-config",
			wantVal: `
name: config-vol
configMap:
  name: my-config`,
		},
		{
			name:          "overwrite existing volume",
			volume:        "foo",
			configMapName: "foo-config",
			keyToPaths: []corev1.KeyToPath{
				{Key: "somekey", Path: "/somepath/"},
			},
			wantVal: `
name: foo
configMap:
  name: foo-config
  items:
    - key: somekey
      path: /somepath/`,
		},
		{
			name:          "with items",
			volume:        "some-config-vol",
			configMapName: "some-config",
			keyToPaths: []corev1.KeyToPath{
				{Key: "key1", Path: "/some/path1"},
				{Key: "key2", Path: "/some/path2"},
			},
			wantVal: `
name: some-config-vol
configMap:
  name: some-config
  items:
    - key: key1
      path: /some/path1
    - key: key2
      path: /some/path2`,
		},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			obj := testObj.Copy()

			// Transform.
			tf, err := SetDaemonSetConfigMapVolumeFunc(tc.volume, tc.configMapName, tc.keyToPaths)
			assert.Nil(t, err)
			err = tf(obj)
			assert.Nil(t, err)

			volumeSelector := fmt.Sprintf("[name=%s]", tc.volume)
			val, err := obj.Pipe(yaml.Lookup("spec", "template", "spec", "volumes", volumeSelector))
			assert.Nil(t, err)

			str, err := val.String()
			assert.Nil(t, err)
			assert.Equal(t, strings.TrimSpace(tc.wantVal), strings.TrimSpace(str))
		})
	}
}

func TestSetDaemonSetSecretVolumeFunc(t *testing.T) {
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
      volumes:
      - name: foo
        secret:
          secretName: mysecret
`)
	assert.Nil(t, err)

	cases := []struct {
		name       string
		volume     string
		secretName string
		keyToPaths []corev1.KeyToPath
		wantVal    string
	}{
		{
			name:       "add new volume",
			volume:     "secret-vol",
			secretName: "my-secret",
			wantVal: `
name: secret-vol
secret:
  secretName: my-secret`,
		},
		{
			name:       "overwrite existing volume",
			volume:     "foo",
			secretName: "foo-secret",
			keyToPaths: []corev1.KeyToPath{
				{Key: "somekey", Path: "/somepath/"},
			},
			wantVal: `
name: foo
secret:
  secretName: foo-secret
  items:
    - key: somekey
      path: /somepath/`,
		},
		{
			name:       "with items",
			volume:     "some-secret-vol",
			secretName: "some-secret",
			keyToPaths: []corev1.KeyToPath{
				{Key: "key1", Path: "/some/path1"},
				{Key: "key2", Path: "/some/path2"},
			},
			wantVal: `
name: some-secret-vol
secret:
  secretName: some-secret
  items:
    - key: key1
      path: /some/path1
    - key: key2
      path: /some/path2`,
		},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			obj := testObj.Copy()

			// Transform.
			tf, err := SetDaemonSetSecretVolumeFunc(tc.volume, tc.secretName, tc.keyToPaths)
			assert.Nil(t, err)
			err = tf(obj)
			assert.Nil(t, err)

			volumeSelector := fmt.Sprintf("[name=%s]", tc.volume)
			val, err := obj.Pipe(yaml.Lookup("spec", "template", "spec", "volumes", volumeSelector))
			assert.Nil(t, err)

			str, err := val.String()
			assert.Nil(t, err)
			assert.Equal(t, strings.TrimSpace(tc.wantVal), strings.TrimSpace(str))
		})
	}
}

func TestSetDaemonSetVolumeMountFunc(t *testing.T) {
	testObj, err := yaml.Parse(`
apiVersion: apps/v1
kind: DaemonSet
metadata:
  name: some-daemonset
spec:
  template:
    spec:
      containers:
      - name: someapp
        image: someapp:v1.5
      - name: someotherapp
        image: some-image:v1.2.3
        volumeMounts:
        - mountPath: /dev/faz
          name: foo
      volumes:
      - name: foo
        secret:
          secretName: mysecret
`)
	assert.Nil(t, err)

	cases := []struct {
		name             string
		container        string
		volName          string
		mountPath        string
		mountPropagation corev1.MountPropagationMode
		wantVal          string
	}{
		{
			name:      "add new volume mount",
			container: "someapp",
			volName:   "app-vol",
			mountPath: "/app/data",
			wantVal: `
name: app-vol
mountPath: /app/data`,
		},
		{
			name:      "overwrite existing volume mount",
			container: "someotherapp",
			volName:   "foo",
			mountPath: "/mnt/foo",
			wantVal: `
mountPath: /mnt/foo
name: foo`,
		},
		{
			name:             "add mount propagation",
			container:        "someotherapp",
			volName:          "foo",
			mountPath:        "/mnt/foo",
			mountPropagation: corev1.MountPropagationBidirectional,
			wantVal: `
mountPath: /mnt/foo
name: foo
mountPropagation: Bidirectional`,
		},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			// Make a copy of the object.
			obj := testObj.Copy()

			// Transform.
			tf := SetDaemonSetVolumeMountFunc(tc.container, tc.volName, tc.mountPath, tc.mountPropagation)
			err = tf(obj)
			assert.Nil(t, err)

			// Query and check the value.
			containerSelector := fmt.Sprintf("[name=%s]", tc.container)
			volMountSelector := fmt.Sprintf("[name=%s]", tc.volName)
			val, err := obj.Pipe(yaml.Lookup("spec", "template", "spec", "containers", containerSelector, "volumeMounts", volMountSelector))
			assert.Nil(t, err)

			str, err := val.String()
			assert.Nil(t, err)
			assert.Equal(t, strings.TrimSpace(tc.wantVal), strings.TrimSpace(str))
		})
	}
}

func TestSetDaemonSetContainerResourceFunc(t *testing.T) {
	testObj, err := yaml.Parse(`
apiVersion: apps/v1
kind: DaemonSet
metadata:
  name: some-daemonset
spec:
  template:
    spec:
      containers:
      - name: someapp
        image: someapp:v1.5
      - name: someotherapp
        image: some-image:v1.2.3
        resources:
          limits:
            cpu: 500m
            memory: 950Mi
          requests:
            memory: 700Mi
`)
	assert.Nil(t, err)

	cases := []struct {
		name         string
		container    string
		resources    corev1.ResourceRequirements
		wantLimits   map[string]string
		wantRequests map[string]string
	}{
		{
			name:      "add new resource",
			container: "someapp",
			resources: corev1.ResourceRequirements{
				Limits: corev1.ResourceList{
					corev1.ResourceMemory: resource.MustParse("1Gi"),
					corev1.ResourceCPU:    resource.MustParse("800m"),
				},
				Requests: corev1.ResourceList{
					corev1.ResourceMemory: resource.MustParse("500Mi"),
					corev1.ResourceCPU:    resource.MustParse("400m"),
				},
			},
			wantLimits: map[string]string{
				"cpu":    "800m",
				"memory": "1Gi",
			},
			wantRequests: map[string]string{
				"cpu":    "400m",
				"memory": "500Mi",
			},
		},
		{
			name:      "overwrite existing resources",
			container: "someotherapp",
			resources: corev1.ResourceRequirements{
				Limits: corev1.ResourceList{
					corev1.ResourceCPU: resource.MustParse("900m"),
				},
				Requests: corev1.ResourceList{
					corev1.ResourceMemory: resource.MustParse("900Mi"),
				},
			},
			wantLimits: map[string]string{
				"cpu":    "900m",
				"memory": "950Mi",
			},
			wantRequests: map[string]string{
				"memory": "900Mi",
			},
		},
		{
			name:      "set limits only",
			container: "someotherapp",
			resources: corev1.ResourceRequirements{
				Limits: corev1.ResourceList{
					corev1.ResourceMemory: resource.MustParse("925Mi"),
				},
			},
			wantLimits: map[string]string{
				"cpu":    "500m",
				"memory": "925Mi",
			},
			wantRequests: map[string]string{
				"memory": "700Mi",
			},
		},
		{
			name:      "set requests only",
			container: "someotherapp",
			resources: corev1.ResourceRequirements{
				Requests: corev1.ResourceList{
					corev1.ResourceMemory: resource.MustParse("925Mi"),
				},
			},
			wantLimits: map[string]string{
				"cpu":    "500m",
				"memory": "950Mi",
			},
			wantRequests: map[string]string{
				"memory": "925Mi",
			},
		},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			// Make a copy of the object.
			obj := testObj.Copy()

			tf := SetDaemonSetContainerResourceFunc(tc.container, tc.resources)
			err = tf(obj)
			assert.Nil(t, err)

			// Query and check the values.
			containerSelector := fmt.Sprintf("[name=%s]", tc.container)

			for key, val := range tc.wantLimits {
				gotLimits, err := obj.Pipe(yaml.Lookup("spec", "template", "spec", "containers", containerSelector, "resources", "limits", key))
				assert.Nil(t, err)
				gotStr, err := gotLimits.String()
				assert.Nil(t, err)
				assert.Equal(t, val, strings.TrimSpace(gotStr))
			}

			for key, val := range tc.wantRequests {
				gotLimits, err := obj.Pipe(yaml.Lookup("spec", "template", "spec", "containers", containerSelector, "resources", "requests", key))
				assert.Nil(t, err)
				gotStr, err := gotLimits.String()
				assert.Nil(t, err)
				assert.Equal(t, val, strings.TrimSpace(gotStr))
			}
		})
	}
}
