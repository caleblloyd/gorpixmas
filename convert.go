package main

import (
	"errors"
	"github.com/caleblloyd/svtracker"
	"io"
	"log"
	"os/exec"
	"time"
)

func Convert(st *svtracker.SvTracker, notWav io.Reader) (io.Reader, error) {
	cmd := exec.Command("which", "ffmpeg")
	cmd.Start()

	done := make(chan error, 1)
	doneRoutine := func(cmd *exec.Cmd) {
		st.Add()
		defer st.Done()
		done <- cmd.Wait()
	}

	go doneRoutine(cmd)
	select {
	case <-st.Term:
		cmd.Process.Kill()
		return nil, errors.New("program terminated")
	case err := <-done:
		if err != nil || !cmd.ProcessState.Success() {
			return nil, errors.New("Not a WAV file. Install ffmpeg to automatically convert to WAV")
		}
	}

	cmd = exec.Command("ffmpeg", "-i", "pipe:0", "-f", "wav", "-")
	cmd.Stdin = notWav
	wavOut, err := cmd.StdoutPipe()
	if err != nil {
		return nil, err
	}
	cmd.Start()

	go doneRoutine(cmd)
	select {
	case err := <-done:
		if err != nil || cmd.ProcessState.Success() {
			return nil, errors.New("FFMPEG was unable to convert file to WAV")
		}
	case <-time.After(time.Second):
		break
	}
	go func() {
		select {
		case <-st.Term:
			cmd.Process.Kill()
		case err := <-done:
			if err != nil || cmd.ProcessState.Success() {
				log.Println("FFMPEG was unable to convert file to WAV")
			}
		}
	}()

	return wavOut, nil
}
