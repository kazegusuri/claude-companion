package main

import (
	"os"
	"os/exec"
)

func main() {
	// This acts as the grandparent claude process
	// It executes a shell script that will then call the wrapper

	if len(os.Args) < 2 {
		os.Exit(1)
	}

	// Execute the shell script passed as first argument
	// Pass all remaining arguments to it
	cmd := exec.Command("/bin/bash", os.Args[1:]...)
	cmd.Stderr = os.Stderr
	cmd.Stdout = os.Stdout
	cmd.Stdin = os.Stdin

	if err := cmd.Run(); err != nil {
		os.Exit(1)
	}
}
