package main

import (
	"flag"
	"fmt"
	"github.com/caleblloyd/svtracker"
	"io"
	"os"
)

func main() {

	flag.Parse()
	if len(flag.Args()) == 0 {
		flag.Usage()
		os.Exit(1)
	}

	st := svtracker.New()
	st.HandleSignals()

	exitError := func (err error){
		os.Stderr.WriteString(fmt.Sprintf("%q\n", err.Error()))
		st.ExitCode = 1
		st.Complete()
	}

	var stdin *os.File
	if flag.Arg(0) == "-"{
		stdin = os.Stdin
	} else {
		f, err := os.Open(flag.Arg(0))
		if err != nil {
			panic(err)
		}
		stdin = f
	}


	delay := 2048
	pipeReader, pipeWriter := io.Pipe()
	delayWriter := NewDelayWriter(pipeWriter, delay)
	stdinReader := io.TeeReader(stdin, delayWriter)
	wavReader, err := Convert(st, stdinReader)
	if (err != nil){
		exitError(err)
	}

	go func(){
		for true {
			data, err := wavReader.ReadSamples(512)
			if (err != nil){
				exitError(err)
			}
			if (data == nil){
				fmt.Println("data nil")
				break
			}
		}
		delayWriter.Flush()
	}()

	go func(){
		err := PlayWav(st, pipeReader)
		if (err != nil){
			exitError(err)
		}
	}()

	st.WaitAndExit()
}
