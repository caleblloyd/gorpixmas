package main

import (
	"math/cmplx"
	"flag"
	"fmt"
	"github.com/caleblloyd/svtracker"
	"io"
	"os"
	"github.com/mjibson/go-dsp/fft"
	"github.com/mjibson/go-dsp/window"
	"strings"
	"github.com/mjibson/go-dsp/wav"
	//"time"
	"time"
)

const bins = 4
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

	exitError := func (err error){
		os.Stderr.WriteString(fmt.Sprintf("%q\n", err.Error()))
		st.ExitCode = 1
		st.Exit()
	}

	var stdin io.Reader
	if flag.Arg(0) == "-"{
		stdin = os.Stdin
	} else {
		f, err := os.Open(flag.Arg(0))
		if err != nil {
			panic(err)
		}
		if (!strings.HasSuffix(strings.ToLower(flag.Arg(0)), ".wav")){
			stdin, err = Convert(st, f)
			if (err != nil) {
				exitError(err)
			}
		} else {
			stdin = f
		}
	}

	pipeReader, pipeWriter := io.Pipe()
	//delayWriter := NewDelayWriter(pipeWriter, 0)
	//stdinReader := io.TeeReader(stdin, delayWriter)
	stdinReader := io.TeeReader(stdin, pipeWriter)
	freqBin, err := NewFreqBin(bins)

	if (err != nil){
		exitError(err)
	}

	go func(){
		wavReader, err := wav.New(pipeReader)
		if (err != nil){
			exitError(err)
		}
		rw := NewRollingWindow(bins, windowSize)
		samplesRemaining := wavReader.Samples
		samplesPerRead := int(wavReader.SampleRate)*int(wavReader.NumChannels)/samplesPerSecond
		durationTotal := time.Duration(0)
		durationPerRead := time.Second / time.Duration(samplesPerSecond)
		boolsCh := make(chan []bool, delay/durationPerRead + 2)
		startTime := time.Now()
		//delayWriter.SetDelay(int(wavReader.ByteRate)/samplesPerSecond)
		for true {
			//fmt.Println(time.Now())
			if (samplesRemaining > 0){
				samplesToRead := samplesPerRead
				if (samplesRemaining < samplesPerRead){
					samplesToRead = samplesRemaining
				}
				samplesRemaining -= samplesToRead
				data, err := wavReader.ReadFloats(samplesPerRead)
				if (err != nil){
					exitError(err)
				}
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
				boolsCh <- rw.CopyBools()
			}
			if (durationTotal >= delay){
				if (len(boolsCh) == 0){
					break
				}
				bools := <-boolsCh
				RevPrint(bools)
			}
			sleepFor := time.Since(startTime) - durationTotal - time.Millisecond
			if sleepFor > time.Duration(0){
				<- time.After(sleepFor)
			}
			durationTotal += durationPerRead
		}
	}()

	go func(){
		err := PlayWav(st, stdinReader)
		if (err != nil){
			exitError(err)
		}
	}()

	st.WaitAndExit()
}
