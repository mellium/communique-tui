package jingle

import "github.com/pion/webrtc/v3"

func createCustomMediaEngine() *webrtc.MediaEngine {
	m := &webrtc.MediaEngine{}

	m.RegisterCodec(webrtc.RTPCodecParameters{
		RTPCodecCapability: webrtc.RTPCodecCapability{webrtc.MimeTypeOpus, 48000, 2, "minptime=10;useinbandfec=1", nil},
		PayloadType:        111,
	}, webrtc.RTPCodecTypeAudio)

	videoRTCPFeedback := []webrtc.RTCPFeedback{{"goog-remb", ""}, {"ccm", "fir"}, {"nack", ""}, {"nack", "pli"}}
	m.RegisterCodec(webrtc.RTPCodecParameters{
		RTPCodecCapability: webrtc.RTPCodecCapability{webrtc.MimeTypeVP8, 90000, 0, "", videoRTCPFeedback},
		PayloadType:        96,
	}, webrtc.RTPCodecTypeVideo)

	return m
}

func createCustomAPI() *webrtc.API {
	return webrtc.NewAPI(
		webrtc.WithMediaEngine(createCustomMediaEngine()),
	)
}
