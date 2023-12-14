package main

import (
	"fmt"
	"sync"
	"time"

	"github.com/pion/webrtc/v3"
)

func rttPingTest(dataChannel *webrtc.DataChannel, idx int) (float64, error) {
	start := time.Now()
	dataChannel.SendText("ping")
	<-boolChanList[idx]
	elapsed := time.Since(start)
	fmt.Printf("PeerConnection %d rtt took %s\n", idx, elapsed)

	return elapsed.Seconds(), nil
}

func rttBatchTest() float64 {
	var wg sync.WaitGroup
	var (
		totalTime float64
		totalTest int
		totalMu   sync.Mutex
	)
	updateTotal := func(elapsedTime float64) {
		totalMu.Lock()
		defer totalMu.Unlock()
		totalTime += elapsedTime
		totalTest++
	}

	for i := 0; i < len(dataChannelList); i++ {
		idx := i
		go func() {
			wg.Add(1)
			defer wg.Done()
			elapsedTime, _ := rttPingTest(dataChannelList[idx], idx)
			updateTotal(elapsedTime)
		}()
	}
	wg.Wait()

	return totalTime / float64(totalTest)
}
