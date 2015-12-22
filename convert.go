package main

import (
	"errors"
	"github.com/caleblloyd/svtracker"
	"github.com/mjibson/go-dsp/wav"
	"io"
	"os/exec"
)

func Convert(st*svtracker.SvTracker, maybeWav io.Reader) (*wav.Wav, error) {
	wavReader, err := wav.New(maybeWav)
	if err != nil {
		cmd := exec.Command("which", "ffmpeg")
		cmd.Start()

		done := make(chan error, 1)
		doneRoutine := func() {
			st.Add()
			defer st.Done()
			done <- cmd.Wait()
		}
		doneRoutine()

		select {
		case <-st.Term:
			cmd.Process.Kill()
			return nil, errors.New("program terminated")
		case err := <-done:
			if err != nil || cmd.ProcessState.Success() {
				return nil, errors.New("Not a WAV file. Install ffmpeg to automatically convert to WAV")
			}
		}

		cmd = exec.Command("ffmpeg", "-i", "pipe:0", "-f", "wav", "-")
		cmd.Stdin = maybeWav
		wavOut, err := cmd.StdoutPipe()
		if (err != nil) {
			return nil, err
		}
		cmd.Start()
		doneRoutine()

		select {
		case <-st.Term:
			cmd.Process.Kill()
			return nil, errors.New("program terminated")
		case err := <-done:
			if err != nil{
				return nil, err
			}
			if (!cmd.ProcessState.Success()) {
				return nil, errors.New("FFMPEG was unable to convert file to WAV")
			}
		}

		wavReader, err = wav.New(wavOut)
		if (err != nil) {
			return nil, err
		}
	}
	return wavReader, nil
}
