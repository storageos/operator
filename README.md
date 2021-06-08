# StorageOS Operator

The StorageOS Operator deploys and configures a StorageOS cluster on Kubernetes.

## Setup/Development

1. Build operator container image with `make docker-build`. Publish or copy
   the container image to an existing k8s cluster to make it available for use
   within the cluster.
2. Generate install manifest file with `make install-manifest`. This will
   generate `storageos-operator.yaml` file.
3. Install the operator with `kubectl create -f storageos-operator.yaml`.

The operator can also be run from outside the cluster with `make run`. Ensure
the CRDs that the operator requires are installed in the cluster before running
it using `make install`.

### Install StorageOS cluster

1. Ensure an etcd cluster is available to be used with StorageOS.
2. Create a secret for the cluster, for example:

```yaml
apiVersion: v1
kind: Secret
metadata:
  name: storageos-api
  namespace: storageos
  labels:
    app: storageos
data:
  # echo -n '<secret>' | base64
  username: c3RvcmFnZW9z
  password: c3RvcmFnZW9z
```

3. Create a `StorageOSCluster` custom resource in the same namespace as the
above secret and refer the secret in the spec:

```yaml
apiVersion: storageos.com/v1
kind: StorageOSCluster
metadata:
  name: storageoscluster-sample
  namespace: storageos
spec:
  secretRefName: storageos-api
  storageClassName: storageos-sc
  kvBackend:
    address: "<etcd-address>"
```

This will create a StorageOS cluster in `storageos` namespace with a
StorageClass `storageos-sc` that can be used to provision StorageOS volumes.

## Testing

Run the unit tests with `make test`.

Run e2e tests with `make e2e`. e2e tests use [kuttl](https://kuttl.dev/).
Install kuttl kubectl plugin before running the e2e tests.
