package cli

import (
	"os"
	"strings"
	"testing"
)

func TestEnrichedEnvironment(t *testing.T) {
	extraVars := map[string]string{}

	extraVars["CHEESE"] = "Challerhocker"
	extraVars["CAMELID"] = "Dromedary"

	subject := EnrichedEnvironment(extraVars)

	if len(subject) != len(os.Environ())+2 {
		t.Log("---------------")
		t.Log("Did not inject the environment correctly")
		t.Logf("Expected length: %d", len(os.Environ())+2)
		t.Logf("Got: %d", len(subject))
		t.Fail()
	}

	for expectedKey, expectedValue := range extraVars {
		var key, value string

		for _, variable := range subject {
			parts := strings.Split(variable, "=")

			if parts[0] == expectedKey {
				key = parts[0]
				value = parts[1]
				break
			}
		}

		if key != expectedKey {
			t.Log("---------------")
			t.Logf("Did not add variable %s to the environment", expectedKey)
			t.Fail()
		} else if value != expectedValue {
			t.Log("---------------")
			t.Logf("Variable %s was added to the environment with an incorrect value", expectedKey)
			t.Logf("Expected: %s", expectedValue)
			t.Logf("Got: %s", value)
			t.Fail()
		}
	}
}
