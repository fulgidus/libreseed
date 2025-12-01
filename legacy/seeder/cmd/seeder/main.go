package main

import (
	"os"

	"github.com/fulgidus/libreseed/seeder/internal/cli"
)

func main() {
	if err := cli.Execute(); err != nil {
		os.Exit(1)
	}
}
