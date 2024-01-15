package main

import (
	"flag"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/pion/webrtc/v3"
	"mellium.im/communique/internal/client/gst"
)

const (
	TURNURLs       string = "turn:turn.slickerius.com:3478"
	TURNUsername   string = "tugasakhir"
	TURNCredential string = "tugasakhirganjil"
)

var (
	peerConnectionList []*webrtc.PeerConnection         = make([]*webrtc.PeerConnection, 0)
	videoTrackList     []*webrtc.TrackLocalStaticSample = make([]*webrtc.TrackLocalStaticSample, 0)
	audioTrackList     []*webrtc.TrackLocalStaticSample = make([]*webrtc.TrackLocalStaticSample, 0)
	dataChannelList    []*webrtc.DataChannel            = make([]*webrtc.DataChannel, 0)
	boolChanList       []chan bool                      = make([]chan bool, 0)
	connWg             sync.WaitGroup
	relayOnly          bool
)

func main() {
	offerAddr := flag.String("offer-address", ":50000", "Address that the Offer HTTP server is hosted on.")
	answerAddr := flag.String("answer-address", "127.0.0.1:60000", "Address that the Answer HTTP server is hosted on.")
	connNumber := flag.Int("conn", 1, "Specify number of peerconnection")
	flag.BoolVar(&relayOnly, "relay", relayOnly, "Use relay only")
	flag.Parse()

	config := webrtc.Configuration{
		ICEServers: []webrtc.ICEServer{
			{
				URLs:       []string{TURNURLs},
				Username:   TURNUsername,
				Credential: TURNCredential,
			},
		},
	}

	gst.GstreamerInit()

	for i := 0; i < *connNumber; i++ {
		createNewPeerConnection(i, config, *answerAddr)
	}

	fmt.Println("Finished initiating PeerConnections, Sending offer...")
	go func() { panic(http.ListenAndServe(*offerAddr, nil)) }()

	for i := 0; i < *connNumber; i++ {
		idx := i
		connWg.Add(1)
		go func() {
			startPeerConnection(peerConnectionList[idx], idx, *answerAddr)
		}()
	}

	connWg.Wait()
	fmt.Println("All PeerConnections connected successfully, starting pipeline...")

	audioPipeline, _ := gst.CreateSendPipeline("audiotest", audioTrackList)
	videoPipeline, _ := gst.CreateSendPipeline("videotest", videoTrackList)
	audioPipeline.Start()
	videoPipeline.Start()

	fmt.Println("Waiting for a while before starting a ping test...")
	time.Sleep(5 * time.Second)

	go func() {
		sigCh := make(chan os.Signal)
		signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)
		select {
		case <-sigCh:
			fmt.Println("Gracefully shutting down all peerConnection...")
			for _, peerConnection := range peerConnectionList {
				peerConnection.Close()
			}
			fmt.Println("done")
			os.Exit(0)
		}
	}()

	for {
		averageTime := rttBatchTest()
		fmt.Printf("Average latency is %fms\n", averageTime)
		time.Sleep(5 * time.Second)
	}
}
