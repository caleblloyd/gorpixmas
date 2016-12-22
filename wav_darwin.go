// +build darwin

package main

import (
	"os/exec"
	"time"
)

const delay = time.Duration(0)

func PlayWavCmd() *exec.Cmd {
	return exec.Command("play", "-t", "wav",  "-")
}
