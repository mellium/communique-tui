package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"sync"
	"time"

	"github.com/pion/rtcp"
	"github.com/pion/webrtc/v3"
)

func signalCandidate(addr string, c *webrtc.ICECandidate, idx int) error {
	payload := []byte(c.ToJSON().Candidate)
	// fmt.Printf("sending candidate: %s\n", string(payload))
	resp, err := http.Post(
		fmt.Sprintf("http://%s/candidate/%d", addr, idx),
		"application/json; charset=utf-8",
		bytes.NewReader(payload),
	)
	if err != nil {
		return err
	}
	return resp.Body.Close()
}

func createNewPeerConnection(idx int, config webrtc.Configuration, answerAddr string) {
	peerConnection, err := webrtc.NewPeerConnection(config)
	if err != nil {
		panic(err)
	}

	var candidatesMux sync.Mutex
	pendingCandidates := make([]*webrtc.ICECandidate, 0)

	opusTrack, err := webrtc.NewTrackLocalStaticSample(webrtc.RTPCodecCapability{MimeType: "audio/opus"}, "audio", "pion1")
	if err != nil {
		panic(err)
	} else if _, err = peerConnection.AddTrack(opusTrack); err != nil {
		panic(err)
	}
	audioTrackList = append(audioTrackList, opusTrack)

	vp8Track, err := webrtc.NewTrackLocalStaticSample(webrtc.RTPCodecCapability{MimeType: "video/vp8"}, "video", "pion2")
	if err != nil {
		panic(err)
	} else if _, err = peerConnection.AddTrack(vp8Track); err != nil {
		panic(err)
	}
	videoTrackList = append(videoTrackList, vp8Track)

	peerConnection.OnICECandidate(func(c *webrtc.ICECandidate) {
		if c == nil {
			return
		}

		if relayOnly && c.Typ != webrtc.ICECandidateTypeRelay {
			return
		}

		if !relayOnly && c.Typ != webrtc.ICECandidateTypeSrflx {
			return
		}

		candidatesMux.Lock()
		defer candidatesMux.Unlock()

		desc := peerConnection.RemoteDescription()
		if desc == nil {
			pendingCandidates = append(pendingCandidates, c)
		} else if onICECandidateErr := signalCandidate(answerAddr, c, idx); onICECandidateErr != nil {
			panic(onICECandidateErr)
		}
	})

	http.HandleFunc(fmt.Sprintf("/candidate/%d", idx), func(w http.ResponseWriter, r *http.Request) {
		candidate, candidateErr := ioutil.ReadAll(r.Body)
		if candidateErr != nil {
			panic(candidateErr)
		}
		if candidateErr := peerConnection.AddICECandidate(webrtc.ICECandidateInit{Candidate: string(candidate)}); candidateErr != nil {
			panic(candidateErr)
		}
	})

	http.HandleFunc(fmt.Sprintf("/sdp/%d", idx), func(w http.ResponseWriter, r *http.Request) {
		sdp := webrtc.SessionDescription{}
		if sdpErr := json.NewDecoder(r.Body).Decode(&sdp); sdpErr != nil {
			panic(sdpErr)
		}

		if sdpErr := peerConnection.SetRemoteDescription(sdp); sdpErr != nil {
			panic(sdpErr)
		}

		candidatesMux.Lock()
		defer candidatesMux.Unlock()

		for _, c := range pendingCandidates {
			if onICECandidateErr := signalCandidate(answerAddr, c, idx); onICECandidateErr != nil {
				panic(onICECandidateErr)
			}
		}

	})

	dataChannel, err := peerConnection.CreateDataChannel("ping", nil)
	if err != nil {
		panic(err)
	}

	dataChannelList = append(dataChannelList, dataChannel)
	boolChanList = append(boolChanList, make(chan bool))

	peerConnection.OnConnectionStateChange(func(s webrtc.PeerConnectionState) {
		fmt.Printf("Peer Connection %d State has changed: %s\n", idx, s.String())

		if s == webrtc.PeerConnectionStateConnected {
			connWg.Done()
		}
	})

	dataChannel.OnMessage(func(msg webrtc.DataChannelMessage) {
		boolChanList[idx] <- true
	})

	peerConnection.OnTrack(func(track *webrtc.TrackRemote, receiver *webrtc.RTPReceiver) {
		go func() {
			ticker := time.NewTicker(time.Second * 3)
			for range ticker.C {
				peerConnection.WriteRTCP([]rtcp.Packet{&rtcp.PictureLossIndication{MediaSSRC: uint32(track.SSRC())}})
				if peerConnection.ConnectionState() != webrtc.PeerConnectionStateConnected {
					return
				}
			}
			buf := make([]byte, 1400)
			for {
				_, _, readErr := track.Read(buf)
				if readErr != nil {
					return
				}
				if peerConnection.ConnectionState() != webrtc.PeerConnectionStateConnected {
					return
				}
			}
		}()
	})

	peerConnectionList = append(peerConnectionList, peerConnection)
}

func startPeerConnection(peerConnection *webrtc.PeerConnection, idx int, answerAddr string) {
	offer, err := peerConnection.CreateOffer(nil)
	if err != nil {
		panic(err)
	}

	if err = peerConnection.SetLocalDescription(offer); err != nil {
		panic(err)
	}

	payload, err := json.Marshal(offer)
	if err != nil {
		panic(err)
	}
	resp, err := http.Post(fmt.Sprintf("http://%s/sdp/%d", answerAddr, idx), "application/json; charset=utf-8", bytes.NewReader(payload)) // nolint:noctx
	if err != nil {
		panic(err)
	} else if err := resp.Body.Close(); err != nil {
		panic(err)
	}
}
