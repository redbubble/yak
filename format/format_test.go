package format

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"github.com/aws/aws-sdk-go/service/sts"
)

var accessKeyId string = "llama"
var secretAccessKey string = "alpaca"
var sessionToken string = "guanaco"

var innerCreds sts.Credentials = sts.Credentials{
	AccessKeyId:     &accessKeyId,
	SecretAccessKey: &secretAccessKey,
	SessionToken:    &sessionToken,
}

var creds sts.AssumeRoleWithSAMLOutput = sts.AssumeRoleWithSAMLOutput{
	Credentials: &innerCreds,
}

func TestEnvCredentials(t *testing.T) {
	expectedLines := []string{
		fmt.Sprintf(`export AWS_ACCESS_KEY_ID=%s`, accessKeyId),
		fmt.Sprintf(`export AWS_SECRET_ACCESS_KEY=%s`, secretAccessKey),
		fmt.Sprintf(`export AWS_SESSION_TOKEN=%s`, sessionToken),
	}

	text, err := Credentials("env", &creds)

	if err != nil {
		t.Log("---------------")
		t.Log("Got an error formatting as \"env\"")
		t.Logf("Error: %v", err)
		t.Fail()
	}

	lines := strings.Split(text, "\n")

	for _, expectedLine := range expectedLines {
		ok := false
		for _, line := range lines {
			if line == expectedLine {
				ok = true
				break
			}
		}

		if !ok {
			t.Log("---------------")
			t.Log("Failed to format credentials as \"env\"")
			t.Logf("Expected content: %v", expectedLines)
			t.Logf("Actual content: %v", lines)
			t.Fail()
			break
		}
	}
}

func TestJsonCredentials(t *testing.T) {
	jsonData, err := Credentials("json", &creds)

	if err != nil {
		t.Log("---------------")
		t.Log("Got an error formatting as \"json\"")
		t.Logf("Error: %v", err)
		t.Fail()
	}

	receivedCreds := sts.Credentials{}

	err = json.Unmarshal([]byte(jsonData), &receivedCreds)

	if err != nil {
		t.Log("---------------")
		t.Log("Got an error formatting as \"json\"")
		t.Logf("Error: %v", err)
		t.Fail()
	}

	if *receivedCreds.AccessKeyId != *innerCreds.AccessKeyId ||
		*receivedCreds.SecretAccessKey != *innerCreds.SecretAccessKey ||
		*receivedCreds.SessionToken != *innerCreds.SessionToken {
		t.Log("---------------")
		t.Log("Failed to format credentials as \"json\"")
		t.Logf("Expected content: %v", innerCreds)
		t.Logf("Actual content: %v", receivedCreds)
		t.Fail()
	}
}

func TestValidateOutputFormat(t *testing.T) {
	if err := ValidateOutputFormat("env"); err != nil {
		t.Log("---------------")
		t.Log("Got an error from ValidateOutputFormat when requesting \"env\"")
		t.Logf("Error: %v", err)
		t.Fail()
	}

	if err := ValidateOutputFormat("json"); err != nil {
		t.Log("---------------")
		t.Log("Got an error from ValidateOutputFormat when requesting \"json\"")
		t.Logf("Error: %v", err)
		t.Fail()
	}

	if ValidateOutputFormat("frankenscript") == nil {
		t.Log("---------------")
		t.Log("Got no error ValidateOutputFormat when requesting an invalid format")
		t.Fail()
	}
}
