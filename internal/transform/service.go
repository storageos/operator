package transform

import (
	"fmt"
	"strconv"

	"github.com/darkowlzz/operator-toolkit/declarative/transform"
	corev1 "k8s.io/api/core/v1"
	kyaml "sigs.k8s.io/kustomize/kyaml/yaml"
)

// SetDefaultServicePortNameFunc sets the default port name of the Service.
func SetDefaultServicePortNameFunc(name string) transform.TransformFunc {
	return func(obj *kyaml.RNode) error {
		portsObj, err := obj.Pipe(
			kyaml.LookupCreate(kyaml.ScalarNode, "spec", "ports"),
		)
		if err != nil {
			return err
		}

		// TODO: Find proper error returned when elements is empty and handle
		// the error appropriately by creating a default port entry.
		ports, err := portsObj.Elements()
		if err != nil {
			return err
		}

		// Set the first port.
		return ports[0].PipeE(
			kyaml.SetField("name", kyaml.NewScalarRNode(name)),
		)
	}
}

// SetServiceTypeFunc sets the type of a Service.
func SetServiceTypeFunc(serviceType corev1.ServiceType) transform.TransformFunc {
	return func(obj *kyaml.RNode) error {
		return obj.PipeE(
			kyaml.LookupCreate(kyaml.ScalarNode, "spec"),
			kyaml.SetField("type", kyaml.NewScalarRNode(string(serviceType))),
		)
	}
}

// SetServiceInternalPortFunc sets the internal port of a Service port.
func SetServiceInternalPortFunc(portName string, port int) transform.TransformFunc {
	return func(obj *kyaml.RNode) error {
		portSelector := fmt.Sprintf("[name=%s]", portName)
		return obj.PipeE(
			kyaml.LookupCreate(kyaml.ScalarNode, "spec", "ports", portSelector),
			kyaml.SetField("port", kyaml.NewScalarRNode(strconv.Itoa(port))),
		)
	}
}

// SetServiceExternalPortFunc sets the external port of a Service port.
func SetServiceExternalPortFunc(portName string, port int) transform.TransformFunc {
	return func(obj *kyaml.RNode) error {
		portSelector := fmt.Sprintf("[name=%s]", portName)
		return obj.PipeE(
			kyaml.LookupCreate(kyaml.ScalarNode, "spec", "ports", portSelector),
			kyaml.SetField("targetPort", kyaml.NewScalarRNode(strconv.Itoa(port))),
		)
	}
}
