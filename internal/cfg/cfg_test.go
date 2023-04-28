package cfg

import (
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestSplitKeyValuesIntoMap(t *testing.T) {
	t.Parallel()

	type testData struct {
		input  string
		result map[string]string
	}

	tests := []testData{
		{
			input: "some_key=value",
			result: map[string]string{
				"some_key": "value",
			},
		},
		{
			input: "key1=value1,key2=value2",
			result: map[string]string{
				"key1": "value1",
				"key2": "value2",
			},
		},
	}

	for _, test := range tests {
		out := SplitKeyValuesIntoMap(test.input)
		if diff := cmp.Diff(test.result, out); diff != "" {
			t.Errorf("SplitKeyValuesIntoMap() mismatch (-want +got):\n%s", diff)
		}
	}
}
