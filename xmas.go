package main

import (
	"math/cmplx"
	"flag"
	"fmt"
	"github.com/caleblloyd/svtracker"
	"io"
	"os"
	"time"
	"github.com/mjibson/go-dsp/fft"
	"github.com/mjibson/go-dsp/window"
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

	pipeReader, pipeWriter := io.Pipe()
	delayWriter := NewDelayWriter(pipeWriter, 0)
	stdinReader := io.TeeReader(stdin, delayWriter)
	bins := 4
	freqBin, err := NewFreqBin(bins)
	samplesPerSecond := 20
	windowSize := 10
	timeDelay := time.Second / time.Duration(samplesPerSecond)
	if (err != nil){
		exitError(err)
	}
	for _, v := range(freqBin.FreqBands){
		fmt.Print(v.Lower)
		fmt.Print(" - ")
		fmt.Println(v.Upper)
	}

	go func(){
		wavReader, err := Convert(st, stdinReader)
		if (err != nil){
			exitError(err)
		}
		samplesPerRead := int(wavReader.SampleRate)*int(wavReader.NumChannels)/samplesPerSecond
		rw := NewRollingWindow(bins, windowSize)
		delayWriter.SetDelay(int(wavReader.ByteRate)/samplesPerSecond)
		for true {
			td := time.After(timeDelay)
			data, err := wavReader.ReadFloats(samplesPerRead)
			convert := make([]float64, len(data))
			for i, v := range(data){
				convert[i] = float64(v)
			}
			window.Apply(convert, window.Bartlett)
			fftOut := fft.FFTReal(convert)
			var magTot float64;
			mag := make([]float64, len(fftOut))
			for i, v := range(fftOut){
				mag[i] = cmplx.Abs(v)
				magTot += mag[i]
			}

			freqSamples := freqBin.BinSamples(mag, int(wavReader.SampleRate))
			rw.SetBoolsAndAddSamples(freqSamples)
			go rw.RevPrint(td)
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
