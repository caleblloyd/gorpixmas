// +build linux

package main

import "os/exec"

func PlayWavCmd() *exec.Cmd {
	return exec.Command("aplay")
}
