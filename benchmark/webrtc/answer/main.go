package main

import (
	"flag"
	"fmt"
	"net/http"

	"github.com/pion/webrtc/v3"
)

const (
	TURNURLs       string = "turn:turn.slickerius.com:3478"
	TURNUsername   string = "tugasakhir"
	TURNCredential string = "tugasakhirganjil"
)

var (
	peerConnectionList []*webrtc.PeerConnection = make([]*webrtc.PeerConnection, 0)
)

func main() {
	offerAddr := flag.String("offer-address", "localhost:50000", "Address that the Offer HTTP server is hosted on.")
	answerAddr := flag.String("answer-address", ":60000", "Address that the Answer HTTP server is hosted on.")
	connNumber := flag.Int("conn", 1, "Specify number of peerconnection")
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

	for i := 0; i < *connNumber; i++ {
		createNewPeerConnection(i, config, *offerAddr)
	}

	fmt.Println("Finished initiating PeerConnections, Waiting for offer...")

	panic(http.ListenAndServe(*answerAddr, nil))
}
