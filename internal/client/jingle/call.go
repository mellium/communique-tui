package jingle

import (
	"encoding/xml"
	"errors"
	"log"
	"strconv"
	"sync"

	"github.com/pion/webrtc/v3"
	"mellium.im/communique/internal/client/gst"
	"mellium.im/xmpp/jid"
	"mellium.im/xmpp/stanza"
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
	api               *webrtc.API
	state             JingleState
	role              JingleRole
	sid               string
	peerConnection    *webrtc.PeerConnection
	receivePipelines  []*gst.ReceivePipeline
	sendPipelines     []*gst.SendPipeline
	debug             *log.Logger
	audioTrack        *webrtc.TrackLocalStaticSample
	videoTrack        *webrtc.TrackLocalStaticSample
	clientJID         jid.JID
	partnerJID        jid.JID
	icePwd            string
	iceUfrag          string
	onIceCandidate    func(ice *webrtc.ICECandidate)
	tempIceCandidates []webrtc.ICECandidateInit
	mu                sync.Mutex
	wg                sync.WaitGroup
}

func New(clientJID jid.JID, onIceCandidate func(ice *webrtc.ICECandidate), debug *log.Logger) *CallClient {
	return &CallClient{
		api:               createCustomAPI(),
		state:             Ended,
		role:              EmptyRole,
		debug:             debug,
		clientJID:         clientJID,
		onIceCandidate:    onIceCandidate,
		tempIceCandidates: []webrtc.ICECandidateInit{},
	}
}

func (c *CallClient) resetClient() {
	if c.peerConnection != nil {
		c.peerConnection.Close()
	}
	c.wg.Wait()

	for _, pipeline := range c.receivePipelines {
		pipeline.Stop()
		pipeline.Free()
	}
	for _, pipeline := range c.sendPipelines {
		pipeline.Stop()
		pipeline.Free()
	}

	c.state = Ended
	c.role = EmptyRole
	c.sid = ""
	c.peerConnection = nil
	c.receivePipelines = nil
	c.sendPipelines = nil
	c.audioTrack = nil
	c.videoTrack = nil
	c.icePwd = ""
	c.iceUfrag = ""
}

func (c *CallClient) WrapJingleMessage(jingleMessage *Jingle) (*IQ, error) {
	if c.state == Ended {
		return nil, errors.New("No jingle session is running")
	}

	return &IQ{
		IQ: stanza.IQ{
			Type: stanza.SetIQ,
			To:   c.partnerJID,
		},
		Jingle: jingleMessage,
	}, nil
}

func (c *CallClient) SetState(state JingleState, role JingleRole, sid string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.state = state
	c.role = role
	c.sid = sid
}

// Return current state synchronously. (state, role, sid)
func (c *CallClient) GetCurrentState() (JingleState, JingleRole, string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.state, c.role, c.sid
}

func (c *CallClient) SetPartnerJid(partnerJid jid.JID) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.partnerJID = partnerJid
}

func (c *CallClient) GetPartnerJid() jid.JID {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.partnerJID
}

func (c *CallClient) StartOutgoingCall(partnerJID jid.JID) (*Jingle, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.state != Ended {
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
	if err = peerConnection.SetLocalDescription(offer); err != nil {
		return nil, err
	}

	// Converting SDP Offer into Jingle
	jingle := FromSDP(offer.SDP)

	// Completing Jingle Attributes
	jingle.Action = "session-initiate"
	jingle.Initiator = c.clientJID.String()
	jingle.SID = randomID()

	// Change CallClient state
	c.state = Pending
	c.role = Initiator
	c.sid = jingle.SID
	c.peerConnection = peerConnection
	c.partnerJID = partnerJID
	c.receivePipelines = []*gst.ReceivePipeline{}
	c.sendPipelines = []*gst.SendPipeline{}
	c.icePwd = jingle.Contents[0].Transport.PWD
	c.iceUfrag = jingle.Contents[0].Transport.UFrag

	return jingle, nil
}

func (c *CallClient) AcceptOutgoingCall(jingle *Jingle) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.state != Pending {
		return errors.New("There's no calling attempt to finalize")
	}
	if c.role != Initiator {
		return errors.New("You are not an initiator")
	}
	if jingle.SID != c.sid {
		return errors.New("Different SID, is this from your intended responder?")
	}

	// Set remote description
	remoteDescription := webrtc.SessionDescription{
		Type: webrtc.SDPTypeAnswer,
		SDP:  jingle.ToSDP(),
	}
	err := c.peerConnection.SetRemoteDescription(remoteDescription)
	if err != nil {
		return err
	}

	// Change CallClient state
	c.state = Active

	return nil
}

func (c *CallClient) AcceptIncomingCall(jingle *Jingle) (*Jingle, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.state != Pending {
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
	err = peerConnection.SetLocalDescription(answer)
	if err != nil {
		return nil, err
	}

	// Pushing ice candidate from temp storage
	for _, ice := range c.tempIceCandidates {
		err = peerConnection.AddICECandidate(ice)
		if err != nil {
			return nil, err
		}
	}
	c.tempIceCandidates = []webrtc.ICECandidateInit{}

	// Converting SDP into jingle
	jingleResponse := FromSDP(answer.SDP)

	// Completing Jingle attributes
	jingleResponse.Action = "session-accept"
	jingleResponse.Responder = c.clientJID.String()
	jingleResponse.SID = jingle.SID

	// Change CallClient State
	c.state = Active
	c.role = Responder
	c.sid = jingleResponse.SID
	c.peerConnection = peerConnection
	c.partnerJID = jid.MustParse(jingle.Initiator)
	c.receivePipelines = []*gst.ReceivePipeline{}
	c.sendPipelines = []*gst.SendPipeline{}
	c.icePwd = jingle.Contents[0].Transport.PWD
	c.iceUfrag = jingle.Contents[0].Transport.UFrag

	return jingleResponse, nil
}

func (c *CallClient) CancelCall() (*Jingle, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.state != Pending {
		return nil, errors.New("No outgoing call to cancel")
	}

	// Generating jingle terminate
	jingle := &Jingle{
		Action: "session-terminate",
		SID:    c.sid,
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

	// Resetting Client
	c.resetClient()

	return jingle, nil
}

func (c *CallClient) TerminateCall() (*Jingle, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.state != Active {
		return nil, errors.New("There's no ongoing call")
	}

	// Generating jingle terminate
	jingle := &Jingle{
		Action: "session-terminate",
		SID:    c.sid,
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

	// Resetting Client
	c.resetClient()

	return jingle, nil
}

func (c *CallClient) RegisterICECandidate(ice *ICECandidate) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.peerConnection == nil {
		c.tempIceCandidates = append(c.tempIceCandidates, webrtc.ICECandidateInit{
			Candidate: ice.toSDP(),
		})
		return nil
	}

	err := c.peerConnection.AddICECandidate(webrtc.ICECandidateInit{
		Candidate: ice.toSDP(),
	})

	return err
}

func (c *CallClient) CreateICECandidateMessage(ice *webrtc.ICECandidate) (*Jingle, error) {
	if c.state == Ended {
		return nil, errors.New("No jingle session currently running")
	}
	c.debug.Printf("New ICE Candidate: %#v", ice)

	iceCandidate := &ICECandidate{
		Component:  strconv.FormatUint(uint64(ice.Component), 10),
		Foundation: ice.Foundation,
		Ip:         ice.Address,
		Port:       strconv.FormatUint(uint64(ice.Port), 10),
		Priority:   strconv.FormatUint(uint64(ice.Priority), 10),
		Protocol:   ice.Protocol.String(),
		Type:       ice.Typ.String(),
	}
	if iceCandidate.Type != "host" {
		iceCandidate.RelAddr = ice.RelatedAddress
		iceCandidate.RelPort = strconv.FormatUint(uint64(ice.RelatedPort), 10)
	}

	content := &Content{
		Creator: "initiator",
		Name:    "0",
		Transport: &ICEUDPTransport{
			PWD:        c.icePwd,
			UFrag:      c.iceUfrag,
			Candidates: []*ICECandidate{iceCandidate},
		},
	}

	jingleMessage := &Jingle{
		Action:   "transport-info",
		SID:      c.sid,
		Contents: []*Content{content},
	}
	if c.role == Initiator {
		jingleMessage.Initiator = c.clientJID.String()
	} else {
		jingleMessage.Responder = c.clientJID.String()
	}

	return jingleMessage, nil
}
