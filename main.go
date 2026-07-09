package main

import (
	"fmt"
	"os"

	"github.com/gangqiang01/EIM-SKILL/cmd"
)

func main() {
	if err := cmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
