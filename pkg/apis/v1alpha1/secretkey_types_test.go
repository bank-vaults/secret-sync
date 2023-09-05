package v1alpha1

import (
	"github.com/stretchr/testify/assert"
	"strings"
	"testing"
)

func TestSecretKey(t *testing.T) {

	for _, tt := range []struct {
		path []string
		key  string
		prop string
	}{
		{}, // empty
		{ // only path
			path: []string{"path", "to"},
		},
		{ // only key
			key: "key",
		},
		{ // only prop
			prop: "prop",
		},
		{ // path and key
			path: []string{"path", "to"},
			key:  "key",
		},
		{ // path and property
			path: []string{"path", "to"},
			prop: "prop",
		},
		{ // key and property
			key:  "key",
			prop: "prop",
		},
		{ // everything
			path: []string{"path", "to"},
			key:  "key",
			prop: "prop",
		},
	} {
		strKey := strings.Join(append(tt.path, tt.key), "/")
		if tt.prop != "" {
			strKey = strKey + "." + tt.prop
		}
		t.Run(strKey, func(t *testing.T) {
			secretKey := SecretKey{Key: strKey}
			assert.Equal(t, tt.path, secretKey.GetPath())
			assert.Equal(t, tt.key, secretKey.GetKey())
			assert.Equal(t, tt.prop, secretKey.GetProperty())
		})
	}
}
