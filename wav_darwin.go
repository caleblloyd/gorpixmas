// +build darwin

package main

import (
	"os/exec"
	"time"
)

func PlayWavCmd() *exec.Cmd {
	return exec.Command("play", "-t", "wav", "-")
}
