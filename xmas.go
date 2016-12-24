package main

import (
	"flag"
	"fmt"
	"github.com/caleblloyd/svtracker"
	"github.com/go-ozzo/ozzo-config"
	"github.com/mjibson/go-dsp/fft"
	"github.com/mjibson/go-dsp/wav"
	"github.com/mjibson/go-dsp/window"
	"io"
	"math/cmplx"
	"os"
	"strings"
	"time"
)

const samplesPerSecond = 16
const windowSize = 4
const threshold = 0.2

func main() {

	flag.Parse()
	if len(flag.Args()) == 0 {
		flag.Usage()
		os.Exit(1)
	}

	st := svtracker.New()
	st.HandleSignals()

	c := config.New()
	c.Load("config.json")
	var gpios []int
	c.Configure(&gpios, "gpios")
	var output string
	c.Configure(&output, "output")
	var delayMs int
	c.Configure(&delayMs, "delayMs")
	delay := time.Duration(delayMs) * time.Millisecond

	exitError := func(err error) {
		os.Stderr.WriteString(fmt.Sprintf("%q\n", err.Error()))
		st.ExitCode = 1
		st.Exit()
	}

	var stdin io.Reader
	if flag.Arg(0) == "-" {
		stdin = os.Stdin
	} else {
		f, err := os.Open(flag.Arg(0))
		if err != nil {
			panic(err)
		}
		if !strings.HasSuffix(strings.ToLower(flag.Arg(0)), ".wav") {
			stdin, err = Convert(st, f)
			if err != nil {
				exitError(err)
			}
		} else {
			stdin = f
		}
	}

	pipeReader, pipeWriter := io.Pipe()
	stdinReader := io.TeeReader(stdin, pipeWriter)
	freqBin, err := NewFreqBin(len(gpios))

	if err != nil {
		exitError(err)
	}

	if output != "console" {
		if err := InitPins(gpios); err != nil {
			exitError(err)
		}
		defer DeinitPins(gpios)
	}

	go func() {
		wavReader, err := wav.New(pipeReader)
		if err != nil {
			exitError(err)
		}
		rw := NewRollingWindow(len(gpios), windowSize)
		samplesRemaining := wavReader.Samples
		samplesPerRead := int(wavReader.SampleRate) * int(wavReader.NumChannels) / samplesPerSecond
		durationTotal := time.Duration(0)
		durationPerRead := time.Second / time.Duration(samplesPerSecond)
		//boolsCh := make(chan []bool, delay/durationPerRead + 2)
		boolsCh := make(chan []bool, 500)
		startTime := time.Now()
		for true {
			if samplesRemaining > 0 {
				samplesToRead := samplesPerRead
				if samplesRemaining < samplesPerRead {
					samplesToRead = samplesRemaining
				}
				samplesRemaining -= samplesToRead
				data, err := wavReader.ReadFloats(samplesPerRead)
				if err != nil {
					exitError(err)
				}
				convert := make([]float64, len(data))
				for i, v := range data {
					convert[i] = float64(v)
				}
				window.Apply(convert, window.Bartlett)
				fftOut := fft.FFTReal(convert)
				var magTot float64
				mag := make([]float64, len(fftOut))
				for i, v := range fftOut {
					mag[i] = cmplx.Abs(v)
					magTot += mag[i]
				}
				freqSamples := freqBin.BinSamples(mag, int(wavReader.SampleRate))
				rw.SetBoolsAndAddSamples(freqSamples)
				boolsCh <- rw.CopyBools()
			}
			if durationTotal >= delay {
				if len(boolsCh) == 0 {
					break
				}
				bools := <-boolsCh
				if output == "console" {
					RevPrint(bools)
				} else {
					SetPins(gpios, bools)
				}
			}
			durationTotal += durationPerRead
			sleepFor := durationTotal - time.Since(startTime) - time.Millisecond
			if sleepFor > time.Duration(0) {
				<-time.After(sleepFor)
			}
		}
	}()

	go func() {
		err := PlayWav(st, stdinReader)
		if err != nil {
			exitError(err)
		}
	}()

	st.WaitAndExit()
}
