package java

import "testing"

func TestJacocoFilterToSpecs(t *testing.T) {
	testCases := []struct {
		name, in, out string
	}{
		{
			name: "class",
			in:   "package.Class",
			out:  "package/Class.class",
		},
		{
			name: "class wildcard",
			in:   "package.Class*",
			out:  "package/Class*.class",
		},
		{
			name: "package wildcard",
			in:   "package.*",
			out:  "package/*.class",
		},
		{
			name: "package recursive wildcard",
			in:   "package.**",
			out:  "package/**/*.class",
		},
		{
			name: "recursive wildcard only",
			in:   "**",
			out:  "**/*.class",
		},
		{
			name: "single wildcard only",
			in:   "*",
			out:  "*.class",
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			got, err := jacocoFilterToSpec(testCase.in)
			if err != nil {
				t.Error(err)
			}
			if got != testCase.out {
				t.Errorf("expected %q got %q", testCase.out, got)
			}
		})
	}
}

func TestJacocoFiltersToZipCommand(t *testing.T) {
	testCases := []struct {
		name               string
		includes, excludes []string
		out                string
	}{
		{
			name:     "implicit wildcard",
			includes: []string{},
			out:      "**/*.class",
		},
		{
			name:     "only include",
			includes: []string{"package/Class.class"},
			out:      "package/Class.class",
		},
		{
			name:     "multiple includes",
			includes: []string{"package/Class.class", "package2/Class.class"},
			out:      "package/Class.class package2/Class.class",
		},
		{
			name:     "excludes",
			includes: []string{"package/**/*.class"},
			excludes: []string{"package/Class.class"},
			out:      "-x package/Class.class package/**/*.class",
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			got := jacocoFiltersToZipCommand(testCase.includes, testCase.excludes)
			if got != testCase.out {
				t.Errorf("expected %q got %q", testCase.out, got)
			}
		})
	}
}
