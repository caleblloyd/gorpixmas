package main

import (
	"errors"
	"fmt"
	"github.com/caleblloyd/svtracker"
	"io"
	"io/ioutil"
)

func PlayWav(st *svtracker.SvTracker, stdin io.Reader) error{
	st.Add()
	defer st.Done()

	cmd := PlayWavCmd()
	cmd.Stdin = stdin
	stderrPipe, _ := cmd.StderrPipe()
	stdoutPipe, _ := cmd.StdoutPipe()
	cmd.Start()

	var stdout, stderr []byte;

	done := make(chan error, 1)
	go func() {
		st.Add()
		defer st.Done()
		stderr, _ = ioutil.ReadAll(stderrPipe)
		stdout, _ = ioutil.ReadAll(stdoutPipe)
		done <- cmd.Wait()
	}()

	select{
	case <- st.Term:
		cmd.Process.Kill()
	case err := <-done:
		if err != nil {
			return errors.New(fmt.Sprintf("wav playing died with error %v\nSTDOUT: %s\nSTDERR: %s", err, string(stderr[:]), string(stdout[:])))
		}
	}
	return nil
}
