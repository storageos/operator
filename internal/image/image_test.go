package image

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSplit(t *testing.T) {
	cases := []struct {
		name       string
		image      string
		wantName   string
		wantTag    string
		wantDigest string
	}{
		{
			name:     "image with tag",
			image:    "hello:v1.0",
			wantName: "hello",
			wantTag:  "v1.0",
		},
		{
			name:       "image with digest",
			image:      "hello@sha256:25a0d4",
			wantName:   "hello",
			wantDigest: "sha256:25a0d4",
		},
		{
			name:     "custom registry",
			image:    "foo.io/example/hello:v1.0",
			wantName: "foo.io/example/hello",
			wantTag:  "v1.0",
		},
		{
			name:     "no tag",
			image:    "hello",
			wantName: "hello",
		},
		{
			name:  "empty",
			image: "",
		},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			gotName, gotTag, gotDigest := Split(tc.image)
			assert.Equal(t, tc.wantName, gotName)
			assert.Equal(t, tc.wantTag, gotTag)
			assert.Equal(t, tc.wantDigest, gotDigest)
		})
	}
}

func TestGetKustomizeImageList(t *testing.T) {
	cases := []struct {
		name        string
		namedImages NamedImages
		wantImages  int
	}{
		{
			name: "mix of image types",
			namedImages: NamedImages{
				"foo": "example.com/foo:v1.0",
				"bar": "xyz.io/bar:v4.9.0",
				"baz": "baz@sha256:25a0d4",
			},
			wantImages: 3,
		},
		{
			name: "empty image value",
			namedImages: NamedImages{
				"foo": "example.com/foo:v1.0",
				"bar": "",
				"baz": "baz@sha256:25a0d4",
			},
			wantImages: 2,
		},
		{
			name:        "empty",
			namedImages: NamedImages{},
			wantImages:  0,
		},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			il := GetKustomizeImageList(tc.namedImages)
			assert.Equal(t, tc.wantImages, len(il))
		})
	}
}
