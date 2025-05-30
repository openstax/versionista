package main

import (
	"fmt"
	"github.com/spf13/viper"
	"os"
)

func CheckError(err error) {
	if err != nil {
		fmt.Printf("Error: %s\n", err)
		os.Exit(1)
	}
}

func Warn(format string, args ...interface{}) {
	fmt.Fprintln(os.Stderr, fmt.Sprintf("[ERROR]: "+format, args...))
}

func main() {
	viper.SetConfigName("versionista")
	viper.SetConfigName(".versionista")
	viper.AddConfigPath("$HOME")
	viper.AddConfigPath(".")
	err := viper.ReadInConfig()
	CheckError(err)

	configureCliCommands()
}
