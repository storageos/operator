# Scorecard Tests

Refer [operatorframework/scorecard][scorecard] for details about the scorecard
tests in general.

### Prerequisites

- operator-sdk binary
- k8s cluster
- OLM in the k8s cluster (run `operator-sdk olm install` to install OLM)

### Preparation

`bundle/` contains an OLM bundle that's used to install the operator in an OLM
enabled cluster. `bundle/tests/scorecard` contains the scorecard configuration
generated with `make bundle`. Update any scorecard configuration by modifying
kustomization config files at `config/scorecard/` and run `make bundle` to
populate `bundle/` with the new configurations. Do not edit `bundle/` by hand.

Before running the scorecard tests, the operator must be installed in the
cluster using the bundle in `bundle/` dir. To create a bundle image, run:

```console
$ make bundle-build BUNDLE_IMG=<namespace>/operator-bundle:test
```

This will copy the files in `bundle/` into a container image in the proper OLM
bundle format. To use the container image and install the bundle, it must be
pullable from a registry, a limitation of how the underlying
[opm][opm-tooling](Operator Package Manager) tool is designed to ensure that
the bundle is usable. Push the bundle image to a registry with:

```console
$ make bundle-push BUNDLE_IMG=<namespace>/operator-bundle:test
```

Run the bundle:

```console
$ operator-sdk run bundle <registry-host>/<namespace>/operator-bundle:test --verbose
$ # operator-sdk run bundle docker.io/storageos/operator-bundle:test --verbose
```

This will run an OLM catalog in the cluster with the given bundle and install
the operator in the bundle. The operator will be installed in the default
namespace.

### Running the tests

Once the operator is installed using the bundle, the scorecard tests can be run
with:

```console
$ operator-sdk scorecard ./bundle --verbose
```

This will run the scorecard tests against the operator and print the test
results:

```console
$ operator-sdk scorecard ./bundle --verbose
DEBU[0000] Debug logging is set
--------------------------------------------------------------------------------
Image:      quay.io/operator-framework/scorecard-test:v1.3.0
Entrypoint: [scorecard-test basic-check-spec]
Labels:
        "suite":"basic"
        "test":"basic-check-spec-test"
Results:
        Name: basic-check-spec
        State: pass



--------------------------------------------------------------------------------
Image:      quay.io/operator-framework/scorecard-test:v1.3.0
Entrypoint: [scorecard-test olm-status-descriptors]
Labels:
        "suite":"olm"
        "test":"olm-status-descriptors-test"
Results:
        Name: olm-status-descriptors
        State: pass

        Log:
                Loaded ClusterServiceVersion: storageosoperator.v2.5.0
                Loaded 1 Custom Resources from alm-examples

...
```

[scorecard]:https://sdk.operatorframework.io/docs/advanced-topics/scorecard/scorecard/
[opm-tooling]:https://github.com/operator-framework/operator-registry/blob/master/docs/design/opm-tooling.md
