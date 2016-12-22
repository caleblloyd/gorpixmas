// +build linux

package main

import (
	"os/exec"
	"time"
)

// aplay default buffer is 500ms
const delay = time.Millisecond * 800

func PlayWavCmd() *exec.Cmd {
	return exec.Command("aplay")
}
