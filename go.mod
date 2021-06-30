module github.com/storageos/operator

go 1.16

require (
	github.com/darkowlzz/operator-toolkit v0.0.0-20210701215849-537f9af6eff6
	github.com/go-logr/logr v0.3.0
	github.com/golang/mock v1.5.0
	github.com/onsi/ginkgo v1.14.1
	github.com/onsi/gomega v1.10.2
	github.com/storageos/go-api/v2 v2.4.0
	github.com/stretchr/testify v1.7.0
	k8s.io/api v0.20.2
	k8s.io/apimachinery v0.20.2
	k8s.io/client-go v0.20.2
	sigs.k8s.io/controller-runtime v0.8.3
	sigs.k8s.io/kustomize/api v0.7.1
	sigs.k8s.io/kustomize/kyaml v0.10.5
	sigs.k8s.io/yaml v1.2.0
)
