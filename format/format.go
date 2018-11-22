package format

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"

	"github.com/aws/aws-sdk-go/service/sts"

	"github.com/redbubble/yak/aws"
)

var outputFormatters map[string]func(*sts.AssumeRoleWithSAMLOutput) (string, error) = map[string]func(*sts.AssumeRoleWithSAMLOutput) (string, error){
	"json": func(creds *sts.AssumeRoleWithSAMLOutput) (string, error) {
		data, err := json.Marshal(creds.Credentials)

		return string(append(data, '\n')), err
	},
	"env": func(creds *sts.AssumeRoleWithSAMLOutput) (string, error) {
		output := bytes.Buffer{}

		var outputFormat string
		if isPowerShell() {
			outputFormat = "$env:%s = \"%s\"\n"
		} else {
			outputFormat = "export %s=%s\n"
		}

		for key, value := range aws.EnvironmentVariables(creds) {
			output.WriteString(fmt.Sprintf(outputFormat, key, value))
		}

		return output.String(), nil
	},
}

func Credentials(format string, creds *sts.AssumeRoleWithSAMLOutput) (string, error) {
	return outputFormatters[format](creds)
}

func ValidateOutputFormat(format string) error {
	if validOutputFormat(format) {
		return nil
	}

	return fmt.Errorf("Invalid output format '%s' specified. Valid output formats: %v", format, validOutputFormats())
}

func isPowerShell() bool {
	_, ok := os.LookupEnv("PSModulePath")
	return ok
}

func validOutputFormat(format string) bool {
	for f, _ := range outputFormatters {
		if format == f {
			return true
		}
	}

	return false
}

func validOutputFormats() []string {
	formats := make([]string, 0, len(outputFormatters))

	for format, _ := range outputFormatters {
		formats = append(formats, format)
	}

	return formats
}
