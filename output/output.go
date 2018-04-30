package output

import (
	"fmt"
	"os"

	"github.com/spf13/viper"
)

func Println(data interface{}) (int, error) {
	if viper.GetBool("cli.quiet") {
		return 0, nil
	}

	return fmt.Println(data)
}

func Printf(format string, objects ...interface{}) (int, error) {
	if viper.GetBool("cli.quiet") {
		return 0, nil
	}

	return fmt.Printf(format, objects...)
}

func ErrorPrintln(data interface{}) (int, error) {
	if viper.GetBool("cli.quiet") {
		return 0, nil
	}

	return fmt.Fprintln(os.Stderr, data)
}

func ErrorPrintf(format string, objects ...interface{}) (int, error) {
	if viper.GetBool("cli.quiet") {
		return 0, nil
	}

	return fmt.Fprintf(os.Stderr, format, objects...)
}
