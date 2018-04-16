package cli

import (
	"fmt"
	"os"
	"os/exec"
)

func EnrichedEnvironment(extraEnv map[string]string) []string {
	env := os.Environ()

	for key, value := range extraEnv {
		env = append(env, fmt.Sprintf("%s=%s", key, value))
	}

	return env
}

func Exec(command []string, environment []string) error {
	cmd := exec.Command(command[0], command[1:]...)
	cmd.Env = environment

	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin

	err := cmd.Start()

	if err != nil {
		return err
	}

	err = cmd.Wait()
	return err
}
