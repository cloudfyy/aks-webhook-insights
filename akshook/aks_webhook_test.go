package akshook

import (
	"testing"
)

func TestMutateRequired(t *testing.T) {
	annotations := map[string]string{
		"appinsights-connstr": "123",
		"appinsights-role":    "456",
	}

	if !mutationRequired(annotations) {
		t.Error("FAIL")
	}

}
