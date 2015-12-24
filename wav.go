package main

import (
	"errors"
	"fmt"
	"github.com/caleblloyd/svtracker"
	"io"
	"io/ioutil"
	"math"
	"os"
	"os/exec"
	"time"
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

type FreqBand struct {
	Lower float64
	Upper float64
}

func NewFreqBand(lower float64, upper float64) *FreqBand{
	return &FreqBand{
		Lower: lower,
		Upper: upper,
	}
}

type FreqBin struct{
	FreqBands []FreqBand
}

func NewFreqBin(numBins int) (*FreqBin, error){
	if (numBins < 1){
		return nil, errors.New("Must have at least 1 frequency bin")
	}
	freqBands := make([]FreqBand, numBins)
	if (numBins == 1){
		freqBands[0] = *NewFreqBand(64, 16384)
	} else {
		// bass and percussion
		freqBands[0] = *NewFreqBand(64, 512)
		// voice and instruments
		// logarithmically range from 512Hz (9^2) to 16 KHz (14^2)
		incr := float64(5.0/numBins)
		for i:=1; i<numBins; i++{
			lower := float64(9.0 + incr * float64(i-1))
			var upper float64
			if (i == numBins - 1){
				upper = 14
			} else {
				upper = lower + incr
			}
			lower = math.Pow(2, lower)
			upper = math.Pow(2, upper)
			freqBands[i] = *NewFreqBand(lower, upper)
		}
	}
	return &FreqBin{
		FreqBands: freqBands,
	}, nil
}

func (fb *FreqBin) BinSamples(samples []float64, samplingFreq int) []float64{
	numSamples := len(samples)
	bins := make([]float64, len(fb.FreqBands))
	sampleI := 0
	for i, v := range(fb.FreqBands){
		for true {
			sampleFreq := float64(sampleI) * (float64(samplingFreq) / float64(numSamples))
			if (sampleFreq >= v.Upper){
//				fmt.Print(v.Lower)
//				fmt.Print(" - ")
//				fmt.Print(v.Upper)
//				fmt.Print(": ")
//				fmt.Println(bins[i])
				break
			}
			if (sampleFreq >= v.Lower){
				bins[i] += samples[sampleI]
			}
			sampleI ++
		}
	}
	return bins
}

func average(xs[]float64)float64 {
	total:=0.0
	for _,v:=range xs {
		total +=v
	}
	return total/float64(len(xs))
}

type RollingWindow struct{
	bins map[int][]float64
	binSize int
	bools []bool
	windowSize int
	windowPos int
}

func NewRollingWindow(binSize int, windowSize int) *RollingWindow{
	bins := make(map[int][]float64, binSize)
	for i:= 0; i<binSize; i++{
		bins[i] = make([]float64, windowSize)
	}
	return &RollingWindow{
		bins: bins,
		binSize: binSize,
		bools: make([]bool, binSize),
		windowSize: windowSize,
	}
}

func (rw *RollingWindow) AddSamples(samples []float64){
	for i, v := range(samples){
		rw.bins[i][rw.windowPos] = v
	}
	rw.windowPos ++
	if (rw.windowPos >= rw.windowSize){
		rw.windowPos = 0
	}
}

func (rw *RollingWindow) SetBoolsAndAddSamples(samples []float64) {
	for i, v := range(samples){
		avg := average(rw.bins[i])
		if (v >= avg){
			rw.bools[i] = true
		} else {
			rw.bools[i] = false
		}
	}
	rw.AddSamples(samples)
}

func (rw *RollingWindow) RevPrint(c <-chan time.Time){
	bools := make([]bool, len(rw.bools))
	copy(bools, rw.bools)
	<- c
	cl := exec.Command("clear")
	cl.Stdout = os.Stdout
	cl.Run()
	for i:=len(bools)-1; i>=0; i--{
		if (rw.bools[i]){
			fmt.Println("1")
		} else {
			fmt.Println(" ")
		}
	}

}