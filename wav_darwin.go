// +build darwin

package main

import "os/exec"

func PlayWavCmd() *exec.Cmd {
	return exec.Command("play", "-t", "wav",  "-")
}
