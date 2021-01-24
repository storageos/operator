package image

import (
	kustomizeimage "sigs.k8s.io/kustomize/api/image"
	kustomizetypes "sigs.k8s.io/kustomize/api/types"
)

// NamedImages is a map of image name and image address/value.
type NamedImages map[string]string

// Split splits the given image into name and tag, without the tag separator.
func Split(image string) (name, tag, digest string) {
	if image == "" {
		return
	}

	nameStr, tagStr := kustomizeimage.Split(image)

	if tagStr != "" {
		// Use the separator identify tag and digest.
		separator := tagStr[:1]

		// kustomize image Split returns tag with the separator. Trim the separator
		// from the tag.
		val := tagStr[1:]

		switch separator {
		case ":":
			return nameStr, val, ""
		case "@":
			return nameStr, "", val
		}
	}
	return nameStr, "", ""
}

// GetKustomizeImageList takes a NamedImages and returns a list of kustomize
// Images. Empty images are ignored.
func GetKustomizeImageList(images NamedImages) []kustomizetypes.Image {
	kImages := []kustomizetypes.Image{}

	for iname, image := range images {
		if image == "" {
			continue
		}
		name, tag, digest := Split(image)
		ki := kustomizetypes.Image{
			Name:    iname,
			NewName: name,
			NewTag:  tag,
			Digest:  digest,
		}
		kImages = append(kImages, ki)
	}

	return kImages
}
