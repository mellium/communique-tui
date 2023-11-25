package jingle

import (
	"errors"
	"log"
	"sync"

	"github.com/pion/webrtc/v3"
	"mellium.im/communique/internal/client/gst"
	"mellium.im/xmpp/jid"
)

type JingleState int

const (
	Ended JingleState = iota
	Pending
	Active
)

type CallClient struct {
	State            JingleState
	SID              string
	RTCClient        *webrtc.PeerConnection
	ReceivePipelines []*gst.ReceivePipeline
	SendPipelines    []*gst.SendPipeline
	debug            *log.Logger
	mu               sync.Mutex
}

func New(debug *log.Logger) *CallClient {
	return &CallClient{
		State: Ended,
		debug: debug,
	}
}

func (c *CallClient) StartCall(initiator *jid.JID) (*Jingle, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.State != Ended {
		return nil, errors.New("Another call is in progress")
	}

	// TODO: Create new PeerConnection and return local SDP in Jingle format
	// Change state to Pending
	return &Jingle{}, nil
}

func (c *CallClient) StartCallFinalize(jingle *Jingle) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.State != Pending {
		return errors.New("There's no calling attempt to finalize")
	}

	// TODO: Process a session-accept response
	// Check the SID, make sure its the same
	return nil
}

func (c *CallClient) AcceptCall(responder *jid.JID, jingle *Jingle) (*Jingle, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.State != Ended {
		return nil, errors.New("Another call is in progress")
	}

	// TODO: Accept a session-initiate request
	// Return SDP Answer in Jingle format
	return nil, nil
}

func (c *CallClient) TerminateCall() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.State == Ended {
		return errors.New("There's no ongoing call")
	}

	// TODO: Terminate call, return Jingle message with session-terminate type
	return nil
}
