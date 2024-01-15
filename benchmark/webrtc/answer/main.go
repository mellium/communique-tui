package main

import (
	"flag"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"

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
	connWg             sync.WaitGroup
	relayOnly          bool
)

func main() {
	offerAddr := flag.String("offer-address", "localhost:50000", "Address that the Offer HTTP server is hosted on.")
	answerAddr := flag.String("answer-address", ":60000", "Address that the Answer HTTP server is hosted on.")
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
		connWg.Add(1)
		createNewPeerConnection(i, config, *offerAddr)
	}

	fmt.Println("Finished initiating PeerConnections, Waiting for offer...")

	go func() { panic(http.ListenAndServe(*answerAddr, nil)) }()

	connWg.Wait()
	fmt.Println("Starting media pipeline...")

	audioPipeline, _ := gst.CreateSendPipeline("audiotest", audioTrackList)
	videoPipeline, _ := gst.CreateSendPipeline("videotest", videoTrackList)
	audioPipeline.Start()
	videoPipeline.Start()

	sigCh := make(chan os.Signal)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)
	// Block forever
	select {
	case <-sigCh:
		fmt.Println("Gracefully shutting down all peerConnection...")
		for _, peerConnection := range peerConnectionList {
			peerConnection.Close()
		}
		fmt.Println("done")
	}
}
