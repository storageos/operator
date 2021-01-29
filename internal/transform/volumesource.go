package transform

import (
	"fmt"
	"strings"

	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/kustomize/kyaml/yaml"
)

const (
	// Volume source names.
	volSrcHostPath  = "hostPath"
	volSrcConfigMap = "configMap"
	volSrcSecret    = "secret"
)

// createHostPathVolumeSource creates a host path volume source kyaml RNode.
func createHostPathVolumeSource(path, pathType string) (*yaml.RNode, error) {
	volSrcString := fmt.Sprintf(`
path: %s
type: %s
`, path, pathType)
	return yaml.Parse(volSrcString)
}

// createKeyValVolumeSource creates a volume source.
// nameRef is the name of the source type, nameVal is the name of the source.
// It returns a kyaml RNode which can be used in further transformation.
func createKeyValVolumeSource(nameRef string, nameVal string, keyToPaths []corev1.KeyToPath) (*yaml.RNode, error) {
	// Construct the volume source.
	volSrcString := fmt.Sprintf(`%s: %s`, nameRef, nameVal)
	volSrc, err := yaml.Parse(volSrcString)
	if err != nil {
		return nil, err
	}

	// Add items if KeyToPath are provided.
	if len(keyToPaths) > 0 {
		// Construct list of key path items.
		itemsStrings := []string{}
		for _, keyToPath := range keyToPaths {
			itemsString := fmt.Sprintf(`
- key: %s
  path: %s
`, keyToPath.Key, keyToPath.Path)
			itemsStrings = append(itemsStrings, itemsString)
		}

		// Join the key path items to form one list.
		itemsListString := strings.Join(itemsStrings, "\n")

		// Parse the list into a RNode.
		itemsList, err := yaml.Parse(itemsListString)
		if err != nil {
			return nil, err
		}

		// Add keyToPath in the volume source items.
		err = volSrc.PipeE(
			yaml.LookupCreate(yaml.ScalarNode),
			yaml.SetField("items", itemsList),
		)
		if err != nil {
			return nil, err
		}
	}

	return volSrc, nil
}
