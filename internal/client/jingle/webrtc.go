package jingle

import (
	"strings"
	"time"

	"github.com/pion/rtcp"
	"github.com/pion/webrtc/v3"
	"mellium.im/communique/internal/client/gst"
)

const (
	TURNURLs       string = "turn:turn.slickerius.com:3478"
	TURNUsername   string = "tugasakhir"
	TURNCredential string = "tugasakhirganjil"
)

func (c *CallClient) onTrackHandler(peerConnection *webrtc.PeerConnection) func(*webrtc.TrackRemote, *webrtc.RTPReceiver) {
	return func(track *webrtc.TrackRemote, receiver *webrtc.RTPReceiver) {
		// Send a PLI on an interval so that the publisher is pushing a keyframe every rtcpPLIInterval
		go func() {
			ticker := time.NewTicker(time.Second * 3)
			for range ticker.C {
				rtcpSendErr := peerConnection.WriteRTCP([]rtcp.Packet{&rtcp.PictureLossIndication{MediaSSRC: uint32(track.SSRC())}})
				if rtcpSendErr != nil {
					c.debug.Println(rtcpSendErr)
				}
				if peerConnection.ConnectionState() != webrtc.PeerConnectionStateConnected {
					c.debug.Printf("Stopping RTCP Send of type %d\n", track.PayloadType())
					return
				}
			}
		}()

		codecName := strings.Split(track.Codec().RTPCodecCapability.MimeType, "/")[1]
		c.debug.Printf("Track has started, of type %d: %s \n", track.PayloadType(), codecName)
		pipeline, _ := gst.CreateReceivePipeline(track.PayloadType(), strings.ToLower(codecName))
		c.ReceivePipelines = append(c.ReceivePipelines, pipeline)

		go func() {
			pipeline.Start()
			c.debug.Printf("Track has stopped, of type %d: %s\n", track.PayloadType(), codecName)
		}()

		buf := make([]byte, 1400)
		for {
			i, _, readErr := track.Read(buf)
			if readErr != nil {
				c.debug.Println(readErr)
			}
			pipeline.Push(buf[:i])
			if peerConnection.ConnectionState() != webrtc.PeerConnectionStateConnected {
				c.debug.Printf("Ending on track handler of type %d: %s\n", track.PayloadType(), codecName)
				return
			}
		}
	}
}

func (c *CallClient) createPeerConnection() (*webrtc.PeerConnection, error) {

	config := webrtc.Configuration{
		ICEServers: []webrtc.ICEServer{
			{
				URLs:       []string{TURNURLs},
				Username:   TURNUsername,
				Credential: TURNCredential,
			},
		},
	}

	peerConnection, err := c.api.NewPeerConnection(config)
	if err != nil {
		return nil, err
	}

	peerConnection.OnTrack(c.onTrackHandler(peerConnection))

	peerConnection.OnICEConnectionStateChange(func(connectionState webrtc.ICEConnectionState) {
		c.debug.Printf("Connection State has changed %s \n", connectionState.String())
	})

	// create audio track
	opusTrack, err := webrtc.NewTrackLocalStaticSample(webrtc.RTPCodecCapability{MimeType: "audio/opus"}, "audio", "pion1")
	if err != nil {
		return nil, err
	} else if _, err = peerConnection.AddTrack(opusTrack); err != nil {
		return nil, err
	}
	c.AudioTrack = opusTrack

	// create video track
	vp8Track, err := webrtc.NewTrackLocalStaticSample(webrtc.RTPCodecCapability{MimeType: "video/vp8"}, "video", "pion2")
	if err != nil {
		return nil, err
	} else if _, err = peerConnection.AddTrack(vp8Track); err != nil {
		return nil, err
	}
	c.VideoTrack = vp8Track

	return peerConnection, nil
}

func (c *CallClient) startTracks() {
	audioPipeline, _ := gst.CreateSendPipeline("opus", []*webrtc.TrackLocalStaticSample{c.AudioTrack})
	videoPipeline, _ := gst.CreateSendPipeline("vp8", []*webrtc.TrackLocalStaticSample{c.VideoTrack})
	c.SendPipelines = append(c.SendPipelines, audioPipeline)
	c.SendPipelines = append(c.SendPipelines, videoPipeline)
	go func() {
		audioPipeline.Start()
	}()
	go func() {
		videoPipeline.Start()
	}()
}
