package jingle

import (
	"encoding/xml"
	"errors"
	"log"
	"sync"

	"github.com/pion/webrtc/v3"
	"mellium.im/communique/internal/client/gst"
	"mellium.im/xmpp/jid"
)

type JingleState int
type JingleRole int

const (
	Ended JingleState = iota
	Pending
	Active
)

const (
	EmptyRole JingleRole = iota
	Initiator
	Responder
)

type CallClient struct {
	State            JingleState
	Role             JingleRole
	SID              string
	RTCClient        *webrtc.PeerConnection
	ReceivePipelines []*gst.ReceivePipeline
	SendPipelines    []*gst.SendPipeline
	debug            *log.Logger
	AudioTrack       *webrtc.TrackLocalStaticSample
	VideoTrack       *webrtc.TrackLocalStaticSample
	mu               sync.Mutex
}

func New(debug *log.Logger) *CallClient {
	return &CallClient{
		State: Ended,
		Role:  EmptyRole,
		debug: debug,
	}
}

func (c *CallClient) GetState() JingleState {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.State
}

func (c *CallClient) GetRole() JingleRole {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.Role
}

func (c *CallClient) StartOutgoingCall(initiator *jid.JID) (*Jingle, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.State != Ended {
		return nil, errors.New("Another call is in progress")
	}

	// Create new peerconnection
	peerConnection, err := c.createPeerConnection()
	if err != nil {
		return nil, err
	}

	// Create offer and gathering ice candidate
	offer, err := peerConnection.CreateOffer(nil)
	if err != nil {
		return nil, err
	}
	gatherComplete := webrtc.GatheringCompletePromise(peerConnection)
	if err = peerConnection.SetLocalDescription(offer); err != nil {
		return nil, err
	}
	<-gatherComplete

	// Converting SDP Offer into Jingle
	localDescription := peerConnection.LocalDescription()
	jingle := FromSDP(localDescription.SDP)

	// Completing Jingle Attributes
	jingle.Action = "session-initiate"
	jingle.Initiator = initiator.String()
	jingle.SID = randomID()

	// Change CallClient state
	c.State = Pending
	c.Role = Initiator
	c.SID = jingle.SID
	c.RTCClient = peerConnection
	c.ReceivePipelines = []*gst.ReceivePipeline{}
	c.SendPipelines = []*gst.SendPipeline{}

	return jingle, nil
}

func (c *CallClient) AcceptOutgoingCall(jingle *Jingle) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.State != Pending {
		return errors.New("There's no calling attempt to finalize")
	}
	if c.Role != Initiator {
		return errors.New("You are not an initiator")
	}
	if jingle.SID != c.SID {
		return errors.New("Different SID, is this from your intended responder?")
	}

	// Set remote description
	remoteDescription := webrtc.SessionDescription{
		Type: webrtc.SDPTypeAnswer,
		SDP:  jingle.ToSDP(),
	}
	err := c.RTCClient.SetRemoteDescription(remoteDescription)
	if err != nil {
		return err
	}

	// Change CallClient state
	c.State = Active

	// Start pushing track buffer
	c.startTracks()

	return nil
}

func (c *CallClient) AcceptIncomingCall(responder *jid.JID, jingle *Jingle) (*Jingle, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.State != Ended {
		return nil, errors.New("Another call is in progress")
	}

	// Create new peerConnection
	peerConnection, err := c.createPeerConnection()
	if err != nil {
		return nil, err
	}

	// Setting remote sdp
	offer := webrtc.SessionDescription{
		Type: webrtc.SDPTypeOffer,
		SDP:  jingle.ToSDP(),
	}
	err = peerConnection.SetRemoteDescription(offer)
	if err != nil {
		return nil, err
	}

	// Create an answer and start ice gathering
	answer, err := peerConnection.CreateAnswer(nil)
	if err != nil {
		return nil, err
	}
	gatherComplete := webrtc.GatheringCompletePromise(peerConnection)
	err = peerConnection.SetLocalDescription(answer)
	if err != nil {
		return nil, err
	}
	<-gatherComplete

	// Converting SDP into jingle
	localDescription := peerConnection.LocalDescription()
	jingleResponse := FromSDP(localDescription.SDP)

	// Completing Jingle attributes
	jingleResponse.Action = "session-accept"
	jingleResponse.Responder = responder.String()
	jingleResponse.SID = jingle.SID

	// Change CallClient State
	c.State = Active
	c.Role = Responder
	c.SID = jingleResponse.SID
	c.ReceivePipelines = []*gst.ReceivePipeline{}
	c.SendPipelines = []*gst.SendPipeline{}

	// Start pushing buffer to track
	c.startTracks()

	return jingleResponse, nil
}

func (c *CallClient) CancelCall() (*Jingle, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.State != Pending {
		return nil, errors.New("No outgoing call to cancel")
	}

	// Closing peerConnection
	c.RTCClient.Close()

	// Generating jingle terminate
	jingle := &Jingle{
		Action: "session-terminate",
		SID:    c.SID,
		Reason: &struct {
			Condition *struct {
				XMLName xml.Name "xml:\",omitempty\""
				Details string   "xml:\",chardata\""
			}
		}{
			Condition: &struct {
				XMLName xml.Name "xml:\",omitempty\""
				Details string   "xml:\",chardata\""
			}{
				XMLName: xml.Name{Local: "cancel"},
			},
		},
	}

	// Cleaning CallClient
	c.State = Ended
	c.Role = EmptyRole
	c.SID = ""
	for _, pipeline := range c.ReceivePipelines {
		pipeline.Stop()
		pipeline.Free()
	}
	for _, pipeline := range c.SendPipelines {
		pipeline.Stop()
		pipeline.Free()
	}

	return jingle, nil
}

func (c *CallClient) TerminateCall() (*Jingle, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.State != Active {
		return nil, errors.New("There's no ongoing call")
	}

	// Closing connection
	c.RTCClient.Close()

	// Generating jingle terminate
	jingle := &Jingle{
		Action: "session-terminate",
		SID:    c.SID,
		Reason: &struct {
			Condition *struct {
				XMLName xml.Name "xml:\",omitempty\""
				Details string   "xml:\",chardata\""
			}
		}{
			Condition: &struct {
				XMLName xml.Name "xml:\",omitempty\""
				Details string   "xml:\",chardata\""
			}{
				XMLName: xml.Name{Local: "success"},
			},
		},
	}

	// Cleaning CallClient
	c.State = Ended
	c.Role = EmptyRole
	c.SID = ""
	for _, pipeline := range c.ReceivePipelines {
		pipeline.Stop()
		pipeline.Free()
	}
	for _, pipeline := range c.SendPipelines {
		pipeline.Stop()
		pipeline.Free()
	}

	return jingle, nil
}
