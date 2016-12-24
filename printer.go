package main

import (
	"fmt"
	"github.com/stianeikeland/go-rpio"
	"os"
	"os/exec"
	"strings"
)

func RevPrint(bools []bool) {
	cl := exec.Command("clear")
	cl.Stdout = os.Stdout
	cl.Run()
	for i := len(bools); i > 0; i-- {
		if bools[i-1] {
			p := strings.Repeat(" ", i-1)
			p += "*" + strings.Repeat("**", len(bools)-i)
			fmt.Println(p)
		} else {
			fmt.Println("")
		}
	}
	p := strings.Repeat(" ", len(bools)-1) + "|"
	fmt.Println(p)
}

func InitPins(gpios []int) error {
	if err := rpio.Open(); err != nil {
		return err
	}
	for _, gpio := range gpios {
		pin := rpio.Pin(gpio)
		pin.Output()
		pin.Low()
	}
	return nil
}

func SetPins(gpios []int, bools []bool) {
	for i, gpio := range gpios {
		pin := rpio.Pin(gpio)
		if bools[i] {
			pin.High()
		} else {
			pin.Low()
		}
	}
}

func DeinitPins(gpios []int) {
	for _, gpio := range gpios {
		pin := rpio.Pin(gpio)
		pin.Low()
	}
	rpio.Close()
}
