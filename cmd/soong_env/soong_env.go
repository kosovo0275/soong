package main

import (
	"flag"
	"fmt"
	"os"

	"android/soong/env"
)

func usage() {
	fmt.Fprintf(os.Stderr, "usage: soong_env env_file\n")
	fmt.Fprintf(os.Stderr, "exits with success if the environment varibles in env_file match\n")
	fmt.Fprintf(os.Stderr, "the current environment\n")
	flag.PrintDefaults()
	os.Exit(2)
}

func main() {
	flag.Parse()

	if flag.NArg() != 1 {
		usage()
	}

	stale, err := env.StaleEnvFile(flag.Arg(0))
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %s\n", err.Error())
		os.Exit(1)
	}

	if stale {
		os.Exit(1)
	}

	os.Exit(0)
}
