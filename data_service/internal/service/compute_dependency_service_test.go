package service

import (
	"reflect"
	"testing"
)

func TestDetectedImports(t *testing.T) {
	tests := []struct {
		name     string
		language string
		script   string
		want     []string
	}{
		{
			name:     "javascript require and import",
			language: "js",
			script:   "const lodash = require('lodash');\nimport '@scope/runtime';\nimport value from 'zod';",
			want:     []string{"@scope/runtime", "lodash", "zod"},
		},
		{
			name:     "python roots and duplicates",
			language: "python",
			script:   "from dateutil import parser\nimport numpy.linalg\nimport dateutil.tz",
			want:     []string{"dateutil", "numpy"},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := detectedImports(test.language, test.script); !reflect.DeepEqual(got, test.want) {
				t.Fatalf("detectedImports() = %#v, want %#v", got, test.want)
			}
		})
	}
}
