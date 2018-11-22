package aws

import (
	"github.com/aws/aws-sdk-go/service/sts"
	"testing"
)

func TestEnvironmentVariables(t *testing.T) {
	accessKeyId := "llama"
	secretAccessKey := "alpaca"
	sessionToken := "guanaco"
	assumedRoleArn := "arn:aws:iam::1234123123:role/sso-vicuña-role"

	creds := sts.AssumeRoleWithSAMLOutput{
		AssumedRoleUser: &sts.AssumedRoleUser{
			Arn: &assumedRoleArn,
		},
		Credentials: &sts.Credentials{
			AccessKeyId:     &accessKeyId,
			SecretAccessKey: &secretAccessKey,
			SessionToken:    &sessionToken,
		},
	}

	subject := EnvironmentVariables(&creds)

	if subject["AWS_ACCESS_KEY_ID"] != accessKeyId {
		t.Log("---------------")
		t.Log("Did not correctly set AWS_ACCESS_KEY_ID")
		t.Logf("Expected: %s", accessKeyId)
		t.Logf("Got: %s", subject["AWS_ACCESS_KEY_ID"])
		t.Fail()
	}

	if subject["AWS_SECRET_ACCESS_KEY"] != secretAccessKey {
		t.Log("---------------")
		t.Log("Did not correctly set AWS_SECRET_ACCESS_KEY")
		t.Logf("Expected: %s", secretAccessKey)
		t.Logf("Got: %s", subject["AWS_SECRET_ACCESS_KEY"])
		t.Fail()
	}

	if subject["AWS_SESSION_TOKEN"] != sessionToken {
		t.Log("---------------")
		t.Log("Did not correctly set AWS_SESSION_TOKEN")
		t.Logf("Expected: %s", sessionToken)
		t.Logf("Got: %s", subject["AWS_SESSION_TOKEN"])
		t.Fail()
	}

	if subject["AWS_METADATA_USER_ARN"] != assumedRoleArn {
		t.Log("---------------")
		t.Log("Did not correctly set AWS_METADATA_USER_ARN")
		t.Logf("Expected: %s", assumedRoleArn)
		t.Logf("Got: %s", subject["AWS_METADATA_USER_ARN"])
		t.Fail()
	}
}
