package command

import (
	"context"
	"io"
	"time"
)

type noiseMaker struct {
	ticker    *time.Ticker
	noiseChan chan []byte
	noiseData []byte
}

func newNoiseMaker(noiseData string) *noiseMaker {
	return &noiseMaker{
		ticker:    time.NewTicker(time.Second),
		noiseChan: make(chan []byte, 10),
		noiseData: []byte(noiseData),
	}
}

func (n *noiseMaker) Run(ctx context.Context) {
	for {
		select {
		case t := <-n.ticker.C:
			data := append(n.noiseData, " "+t.String()+"\n"...)
			n.noiseChan <- data
		case <-ctx.Done():
			break
		}
	}
}

func (n *noiseMaker) Read(p []byte) (size int, err error) {
	data, ok := <-n.noiseChan
	if !ok {
		return 0, io.EOF
	}
	copy(p, data)
	return len(p), nil
}

func (n *noiseMaker) Close() error {
	n.ticker.Stop()
	close(n.noiseChan)
	return nil
}
