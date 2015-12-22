package main

import(
	"errors"
	"io"
	"sync"
)

type DelayWriter struct {
	io.Writer
	buffer []byte
	delay int
	mutex *sync.Mutex
	readPos int
	writePos int
	written int
}

func NewDelayWriter(writer io.Writer, delay int) *DelayWriter{
	return &DelayWriter{
		Writer: writer,
		buffer: make([]byte, delay),
		delay: delay,
		mutex: &sync.Mutex{},
	}
}

func (dw *DelayWriter) Flush() (int, error) {
	dw.mutex.Lock()
	defer dw.mutex.Unlock()
	n, err := dw.flushBuffer(dw.delay)
	return n, err
}

func (dw *DelayWriter) flushBuffer(numBytes int) (int, error) {
	if (numBytes > dw.written){
		numBytes = dw.written
	}
	numBytesTotal := numBytes
	var sl []byte
	if (dw.readPos + numBytes > dw.delay){
		sl = dw.buffer[dw.readPos : ]
		dw.readPos = 0
		written := len(sl)
		numBytes -= written
	}
	sl = append(sl, dw.buffer[dw.readPos : dw.readPos + numBytes]...)
	n, err := dw.Writer.Write(sl)
	dw.readPos += numBytes
	if (dw.readPos == dw.delay){
		dw.readPos = 0
	}
	dw.written -= numBytesTotal
	return n, err
}

func (dw *DelayWriter) writeBuffer(p []byte) (int, error) {
	numBytes := len(p)
	numBytesTotal := numBytes
	if (numBytes > dw.delay){
		return 0, errors.New("writing more bytes than buffer")
	}
	var pStart int
	if (dw.writePos + numBytes > dw.delay){
		pStart = dw.delay - dw.writePos
		copy(dw.buffer[dw.writePos : ], p[0 : pStart])
		dw.writePos = 0
		numBytes -= pStart
		dw.written += pStart
	}
	copy(dw.buffer[dw.writePos : dw.writePos + numBytes], p[pStart:])
	dw.writePos += numBytes
	if dw.writePos == dw.delay{
		dw.writePos = 0
	}
	dw.written += numBytes
	return numBytesTotal, nil
}

func (dw *DelayWriter) Write(p []byte) (int, error) {
	dw.mutex.Lock()
	defer dw.mutex.Unlock()
	numBytes := len(p)
	var n, nTotal, pStart int
	var err error
	for numBytes > 0 {
		var numBytesLoop int
		if (numBytes > dw.delay){
			numBytesLoop = dw.delay
		} else {
			numBytesLoop = numBytes
		}
		if dw.written + numBytesLoop > dw.delay{
			dw.flushBuffer(dw.written + numBytesLoop - dw.delay)
		}
		n, err = dw.writeBuffer(p[pStart : pStart + numBytesLoop])
		nTotal += n
		if (err != nil){
			return nTotal, err
		}
		pStart += numBytesLoop
		numBytes -= numBytesLoop
	}
	return nTotal, err
}
