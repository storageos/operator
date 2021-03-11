package transform

import (
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	kyaml "sigs.k8s.io/kustomize/kyaml/yaml"
)

func TestSetPodTemplateEnvVarStringFunc(t *testing.T) {
	testObj, err := kyaml.Parse(`
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
			tf := SetPodTemplateContainerEnvVarStringFunc(tc.container, tc.key, tc.val)
			err = tf(obj)
			assert.Nil(t, err)

			// Query and check the value.
			containerSelector := fmt.Sprintf("[name=%s]", tc.container)
			envVarSelector := fmt.Sprintf("[name=%s]", tc.key)
			val, err := obj.Pipe(kyaml.Lookup("spec", "template", "spec", "containers", containerSelector, "env", envVarSelector, "value"))
			assert.Nil(t, err)

			str, err := val.String()
			assert.Nil(t, err)
			assert.Equal(t, tc.val, strings.TrimSpace(str))
		})
	}
}

func TestSetPodTemplateEnvVarValueFromSecretFunc(t *testing.T) {
	testObj, err := kyaml.Parse(`
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
  key: username
  name: init-secret`,
		},
		{
			name:       "overwrite existing env var",
			container:  "someotherapp",
			key:        "SECRET_USER",
			secretName: "creds",
			secretKey:  "user",
			wantVal: `
secretKeyRef:
  key: user
  name: creds`,
		},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			// Make a copy of the object.
			obj := testObj.Copy()
			tf := SetPodTemplateContainerEnvVarValueFromSecretFunc(tc.container, tc.key, tc.secretName, tc.secretKey)
			err = tf(obj)
			assert.Nil(t, err)

			containerSelector := fmt.Sprintf("[name=%s]", tc.container)
			envVarSelector := fmt.Sprintf("[name=%s]", tc.key)
			val, err := obj.Pipe(kyaml.Lookup("spec", "template", "spec", "containers", containerSelector, "env", envVarSelector, "valueFrom"))
			assert.Nil(t, err)

			str, err := val.String()
			assert.Nil(t, err)
			assert.Equal(t, strings.TrimSpace(tc.wantVal), strings.TrimSpace(str))
		})
	}
}

func TestSetPodTemplateEnvVarValueFromFieldFunc(t *testing.T) {
	testObj, err := kyaml.Parse(`
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
			tf := SetPodTemplateContainerEnvVarValueFromFieldFunc(tc.container, tc.key, tc.fieldPath)
			err = tf(obj)
			assert.Nil(t, err)

			containerSelector := fmt.Sprintf("[name=%s]", tc.container)
			envVarSelector := fmt.Sprintf("[name=%s]", tc.key)
			val, err := obj.Pipe(kyaml.Lookup("spec", "template", "spec", "containers", containerSelector, "env", envVarSelector, "valueFrom"))
			assert.Nil(t, err)

			str, err := val.String()
			assert.Nil(t, err)
			assert.Equal(t, strings.TrimSpace(tc.wantVal), strings.TrimSpace(str))
		})
	}
}

func TestSetPodTemplateHostPathVolumeFunc(t *testing.T) {
	testObj, err := kyaml.Parse(`
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
		pathType corev1.HostPathType
		wantVal  string
	}{
		{
			name:     "new volume",
			volume:   "somedir",
			path:     "/var/lib/foo",
			pathType: corev1.HostPathDirectory,
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
			pathType: corev1.HostPathFile,
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
			pathType: corev1.HostPathFile,
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
  type: ""
`,
		},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			obj := testObj.Copy()

			// Transform.
			tf := SetPodTemplateHostPathVolumeFunc(tc.volume, tc.path, &tc.pathType)
			err = tf(obj)
			assert.Nil(t, err)

			volumeSelector := fmt.Sprintf("[name=%s]", tc.volume)
			val, err := obj.Pipe(kyaml.Lookup("spec", "template", "spec", "volumes", volumeSelector))
			assert.Nil(t, err)

			str, err := val.String()
			assert.Nil(t, err)
			assert.Equal(t, strings.TrimSpace(tc.wantVal), strings.TrimSpace(str))
		})
	}
}

func TestSetPodTemplateConfigMapVolumeFunc(t *testing.T) {
	testObj, err := kyaml.Parse(`
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
  items:
    - key: somekey
      path: /somepath/
  name: foo-config`,
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
  items:
    - key: key1
      path: /some/path1
    - key: key2
      path: /some/path2
  name: some-config`,
		},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			obj := testObj.Copy()

			// Transform.
			tf := SetPodTemplateConfigMapVolumeFunc(tc.volume, tc.configMapName, tc.keyToPaths)
			err = tf(obj)
			assert.Nil(t, err)

			volumeSelector := fmt.Sprintf("[name=%s]", tc.volume)
			val, err := obj.Pipe(kyaml.Lookup("spec", "template", "spec", "volumes", volumeSelector))
			assert.Nil(t, err)

			str, err := val.String()
			assert.Nil(t, err)
			assert.Equal(t, strings.TrimSpace(tc.wantVal), strings.TrimSpace(str))
		})
	}
}

func TestSetPodTemplateSecretVolumeFunc(t *testing.T) {
	testObj, err := kyaml.Parse(`
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
  items:
    - key: somekey
      path: /somepath/
  secretName: foo-secret`,
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
  items:
    - key: key1
      path: /some/path1
    - key: key2
      path: /some/path2
  secretName: some-secret`,
		},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			obj := testObj.Copy()

			// Transform.
			tf := SetPodTemplateSecretVolumeFunc(tc.volume, tc.secretName, tc.keyToPaths)
			err = tf(obj)
			assert.Nil(t, err)

			volumeSelector := fmt.Sprintf("[name=%s]", tc.volume)
			val, err := obj.Pipe(kyaml.Lookup("spec", "template", "spec", "volumes", volumeSelector))
			assert.Nil(t, err)

			str, err := val.String()
			assert.Nil(t, err)
			assert.Equal(t, strings.TrimSpace(tc.wantVal), strings.TrimSpace(str))
		})
	}
}

func TestSetPodTemplateVolumeMountFunc(t *testing.T) {
	testObj, err := kyaml.Parse(`
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
			tf := SetPodTemplateVolumeMountFunc(tc.container, tc.volName, tc.mountPath, tc.mountPropagation)
			err = tf(obj)
			assert.Nil(t, err)

			// Query and check the value.
			containerSelector := fmt.Sprintf("[name=%s]", tc.container)
			volMountSelector := fmt.Sprintf("[name=%s]", tc.volName)
			val, err := obj.Pipe(kyaml.Lookup("spec", "template", "spec", "containers", containerSelector, "volumeMounts", volMountSelector))
			assert.Nil(t, err)

			str, err := val.String()
			assert.Nil(t, err)
			assert.Equal(t, strings.TrimSpace(tc.wantVal), strings.TrimSpace(str))
		})
	}
}

func TestSetPodTemplateContainerResourceFunc(t *testing.T) {
	testObj, err := kyaml.Parse(`
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
				"cpu": "900m",
			},
			wantRequests: map[string]string{
				"memory": "900Mi",
			},
		},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			// Make a copy of the object.
			obj := testObj.Copy()

			tf := SetPodTemplateContainerResourceFunc(tc.container, tc.resources)
			err = tf(obj)
			assert.Nil(t, err)

			// Query and check the values.
			containerSelector := fmt.Sprintf("[name=%s]", tc.container)

			for key, val := range tc.wantLimits {
				gotLimits, err := obj.Pipe(kyaml.Lookup("spec", "template", "spec", "containers", containerSelector, "resources", "limits", key))
				assert.Nil(t, err)
				gotStr, err := gotLimits.String()
				assert.Nil(t, err)
				assert.Equal(t, val, strings.TrimSpace(gotStr))
			}

			for key, val := range tc.wantRequests {
				gotLimits, err := obj.Pipe(kyaml.Lookup("spec", "template", "spec", "containers", containerSelector, "resources", "requests", key))
				assert.Nil(t, err)
				gotStr, err := gotLimits.String()
				assert.Nil(t, err)
				assert.Equal(t, val, strings.TrimSpace(gotStr))
			}
		})
	}
}

func TestSetPodTemplateTolerationsFunc(t *testing.T) {
	testObj, err := kyaml.Parse(`
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
`)
	assert.Nil(t, err)

	testObjWithTolerations, err := kyaml.Parse(`
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
      tolerations:
        - key: some-xyz
          operator: Exists
`)
	assert.Nil(t, err)

	cases := []struct {
		name            string
		object          *kyaml.RNode
		tolerations     []corev1.Toleration
		wantTolerations string
		wantErr         bool
	}{
		{
			name:   "add tolerations",
			object: testObj,
			tolerations: []corev1.Toleration{
				{
					Key:      "some-toleration",
					Operator: corev1.TolerationOpEqual,
					Value:    "foo",
					Effect:   corev1.TaintEffectNoExecute,
				},
			},
			wantTolerations: `
- effect: NoExecute
  key: some-toleration
  operator: Equal
  value: foo
`,
		},
		{
			name:   "overwrite tolerations",
			object: testObjWithTolerations,
			tolerations: []corev1.Toleration{
				{
					Key:      "some-toleration",
					Operator: corev1.TolerationOpEqual,
					Value:    "foo",
					Effect:   corev1.TaintEffectNoExecute,
				},
				{
					Key:      "someother-toleration",
					Operator: corev1.TolerationOpExists,
				},
			},
			wantTolerations: `
- effect: NoExecute
  key: some-toleration
  operator: Equal
  value: foo
- key: someother-toleration
  operator: Exists`,
		},
		{
			name:   "invalid value",
			object: testObj,
			tolerations: []corev1.Toleration{
				{
					Key:      "some-toleration",
					Operator: corev1.TolerationOpExists,
					Value:    "foo",
				},
			},
			wantErr: true,
		},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			// Make a copy of the object.
			obj := tc.object.Copy()

			tf := SetPodTemplateTolerationFunc(tc.tolerations)
			err = tf(obj)
			if !tc.wantErr {
				assert.Nil(t, err)

				// Query and check the result.
				gotTolerations, err := obj.Pipe(kyaml.Lookup("spec", "template", "spec", "tolerations"))
				assert.Nil(t, err)
				gotStr, err := gotTolerations.String()
				assert.Nil(t, err)
				assert.Equal(t, strings.TrimSpace(tc.wantTolerations), strings.TrimSpace(gotStr))
			} else {
				assert.NotNil(t, err)
			}
		})
	}
}

func TestSetPodTemplateNodeSelectorTermsFunc(t *testing.T) {
	testObj, err := kyaml.Parse(`
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
`)
	assert.Nil(t, err)

	testObjWithAffinity, err := kyaml.Parse(`
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
      affinity:
        nodeAffinity:
          requiredDuringSchedulingIgnoredDuringExecution:
            nodeSelectorTerms:
              - matchExpressions:
                  - key: somekey
                    operator: In
                    values:
                    - someval
`)
	assert.Nil(t, err)

	cases := []struct {
		name              string
		object            *kyaml.RNode
		nodeSelectorTerms []corev1.NodeSelectorTerm
		wantNodeAffinity  string
	}{
		{
			name:   "add selector",
			object: testObj,
			nodeSelectorTerms: []corev1.NodeSelectorTerm{
				{
					MatchExpressions: []corev1.NodeSelectorRequirement{
						{
							Key:      "foo",
							Operator: corev1.NodeSelectorOpIn,
							Values:   []string{"baz"},
						},
					},
				},
			},
			wantNodeAffinity: `
nodeAffinity:
  requiredDuringSchedulingIgnoredDuringExecution:
    nodeSelectorTerms:
      - matchExpressions:
          - key: foo
            operator: In
            values:
              - baz`,
		},
		{
			name:   "overwrite selector",
			object: testObjWithAffinity,
			nodeSelectorTerms: []corev1.NodeSelectorTerm{
				{
					MatchExpressions: []corev1.NodeSelectorRequirement{
						{
							Key:      "foo",
							Operator: corev1.NodeSelectorOpIn,
							Values:   []string{"baz"},
						},
					},
				},
				{
					MatchFields: []corev1.NodeSelectorRequirement{
						{
							Key:      "baz",
							Operator: corev1.NodeSelectorOpExists,
						},
					},
				},
			},
			wantNodeAffinity: `
nodeAffinity:
  requiredDuringSchedulingIgnoredDuringExecution:
    nodeSelectorTerms:
      - matchExpressions:
          - key: foo
            operator: In
            values:
              - baz
      - matchFields:
          - key: baz
            operator: Exists`,
		},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			// Make a copy of the object.
			obj := tc.object.Copy()

			tf := SetPodTemplateNodeSelectorTermsFunc(tc.nodeSelectorTerms)
			err = tf(obj)
			assert.Nil(t, err)

			// Query and check the result.
			gotAffinity, err := obj.Pipe(kyaml.Lookup("spec", "template", "spec", "affinity"))
			assert.Nil(t, err)
			gotStr, err := gotAffinity.String()
			assert.Nil(t, err)
			assert.Equal(t, strings.TrimSpace(tc.wantNodeAffinity), strings.TrimSpace(gotStr))
		})
	}
}

func TestAppendPodTemplateContainerArgsFunc(t *testing.T) {
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
			tf := AppendPodTemplateContainerArgsFunc(tc.container, tc.args)
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
