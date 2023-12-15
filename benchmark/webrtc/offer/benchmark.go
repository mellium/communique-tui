package main

import (
	"sync"
	"time"

	"github.com/pion/webrtc/v3"
)

func rttPingTest(dataChannel *webrtc.DataChannel, idx int) (int64, error) {
	start := time.Now()
	dataChannel.SendText("ping")
	<-boolChanList[idx]
	elapsed := time.Since(start)
	// fmt.Printf("PeerConnection %d rtt took %s\n", idx, elapsed)

	return elapsed.Milliseconds(), nil
}

func rttBatchTest() float64 {
	var wg sync.WaitGroup
	var (
		totalTime int64
		totalTest int
		totalMu   sync.Mutex
	)
	updateTotal := func(elapsedTime int64) {
		totalMu.Lock()
		defer totalMu.Unlock()
		totalTime += elapsedTime
		totalTest++
	}

	for i := 0; i < len(dataChannelList); i++ {
		idx := i
		wg.Add(1)
		go func() {
			elapsedTime, _ := rttPingTest(dataChannelList[idx], idx)
			updateTotal(elapsedTime)
			wg.Done()
		}()
	}
	wg.Wait()

	return float64(totalTime) / float64(totalTest)
}
