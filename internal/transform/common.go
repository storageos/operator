package transform

import (
	"github.com/darkowlzz/operator-toolkit/declarative/transform"
	kyaml "sigs.k8s.io/kustomize/kyaml/yaml"
	"sigs.k8s.io/yaml"
)

const (
	// Container types.
	containerTypeMain = "containers"
	containerTypeInit = "initContainers"

	// Field names of the environment variables.
	envVarValue     = "value"
	envVarValueFrom = "valueFrom"

	// Field names in volume mounts.
	volMountPath        = "mountPath"
	volMountPropagation = "mountPropagation"

	// Field name of resource requirements.
	resourcesField = "resources"

	// Field name of tolerations.
	tolerationsField = "tolerations"

	// Field name of node selector.
	nodeSelectorField = "requiredDuringSchedulingIgnoredDuringExecution"
)

// goToRNode converts any go type into a kyaml RNode.
func goToRNode(goObj interface{}) (*kyaml.RNode, error) {
	// Convert go typed resource to yaml.
	// NOTE: Using sigs.k8s.io/yaml for correct decoding of the values.
	// Using kyaml's Marshal doesn't decode with proper values.
	//
	// Following is an example result of using kyaml marshal:
	//   limits:
	//	   cpu:
	//		   format: DecimalSI
	//	   memory:
	//		   format: BinarySI
	//
	// Following is an example result of using sigs.k8s.io/yaml marshal:
	//   limits:
	//     cpu: 200m
	//     memory: 500Mi
	objBytes, err := yaml.Marshal(goObj)
	if err != nil {
		return nil, err
	}

	obj, err := kyaml.Parse(string(objBytes))
	if err != nil {
		return nil, err
	}

	return obj, nil
}

// SetScalarNodeFunc sets a scalar node field with the given value at the given
// path.
func SetScalarNodeFunc(valField string, value *kyaml.RNode, path ...string) transform.TransformFunc {
	return func(obj *kyaml.RNode) error {
		return obj.PipeE(
			kyaml.LookupCreate(kyaml.ScalarNode, path...),
			kyaml.SetField(valField, value),
		)
	}
}

// SetScalarNodeStringValueFunc sets a scalar node field with the given string
// value at the given path.
func SetScalarNodeStringValueFunc(valField string, value string, path ...string) transform.TransformFunc {
	return SetScalarNodeFunc(valField, kyaml.NewScalarRNode(value), path...)
}

// SetMetadataNameFunc sets the metadata name of a given resource.
func SetMetadataNameFunc(name string) transform.TransformFunc {
	return SetScalarNodeStringValueFunc("name", name, "metadata")
}

// SetVolumeMountFunc sets the volume mount at a given path.
func SetVolumeMountFunc(value *kyaml.RNode, mountPropagationValue *kyaml.RNode, path ...string) transform.TransformFunc {
	return func(obj *kyaml.RNode) error {
		// Add mount path.
		tf := SetScalarNodeFunc(volMountPath, value, path...)
		if err := tf(obj); err != nil {
			return err
		}
		// Add mount propagation if provided.
		if mountPropagationValue != nil {
			tf := SetScalarNodeFunc(volMountPropagation, mountPropagationValue, path...)
			if err := tf(obj); err != nil {
				return err
			}
		}
		return nil
	}
}
