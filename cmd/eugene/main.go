package main

import (
	"fmt"
	"os"

	"github.com/kolah/eugene/internal/cli"
)

func main() {
	cmd := cli.RootCmd()
	err := cmd.Execute()
	if err != nil {
		fmt.Println(err.Error())
		os.Exit(1)
	}
}
