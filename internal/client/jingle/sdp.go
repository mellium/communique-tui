package jingle

import "strings"

type attributeField struct {
	name      string
	value     string
	extension string
}

// Accept attribute value (a=<attribute-value>)
func attributeParse(aField string) *attributeField {
	ret := &attributeField{}
	firstSpace := strings.Index(aField, " ")
	attribute := aField
	if firstSpace != -1 {
		attribute = aField[:firstSpace]
		ret.extension = aField[firstSpace+1:]
	}
	attributeSplit := strings.Split(attribute, ":")
	ret.name = attributeSplit[0]
	if len(attributeSplit) == 2 {
		ret.value = attributeSplit[1]
	}
	return ret
}

// Convert to sdp format of a=<attribute-field>
func (a *attributeField) toString() string {
	ret := a.name
	if a.value != "" {
		ret += ":" + a.value
		if a.extension != "" {
			ret += " " + a.extension
		}
	}
	return ret
}

type mediaField struct {
	media string
	ids   []string
}

// Accept media value (m=<media-value>)
func mediaParse(mField string) *mediaField {
	ret := &mediaField{
		ids: []string{},
	}
	mediaSplit := strings.Split(mField, " ")
	ret.media = mediaSplit[0]
	ret.ids = append(ret.ids, mediaSplit[3:]...)
	return ret
}

// Convert to sdp format of m=<media-field>
func (m *mediaField) toString() string {
	ret := m.media + " 9 UDP/TLS/RTP/SAVPF"
	for _, id := range m.ids {
		ret += " " + id
	}
	return ret
}

// Create Jingle message from SDP Value
func FromSDP(sdp string) *Jingle {
	j := &Jingle{
		Group: &struct {
			Semantics string "xml:\"semantics,attr,omitempty\""
			Contents  []struct {
				Name string "xml:\"name,attr,omitempty\""
			} "xml:\"content,omitempty\""
		}{
			Contents: []struct {
				Name string "xml:\"name,attr,omitempty\""
			}{},
		},
		Contents: []*Content{},
	}
	var iceCandidates []*ICECandidate
	var fingerprint *FingerPrint
	sdplines := strings.Split(sdp, "\r\n")
	i := 0
	for i < len(sdplines) {
		sdpline := sdplines[i]
		if sdpline == "" {
			i++
			continue
		}
		sdpType := sdpline[0:1]
		sdpValue := sdpline[2:]
		switch sdpType {
		case "a":
			attribute := attributeParse(sdpValue)
			switch attribute.name {
			case "group":
				j.Group.Semantics = attribute.value
				for _, content := range strings.Split(attribute.extension, " ") {
					j.Group.Contents = append(j.Group.Contents, struct {
						Name string "xml:\"name,attr,omitempty\""
					}{Name: content})
				}
			case "fingerprint":
				if fingerprint == nil {
					fingerprint = &FingerPrint{}
				}
				fingerprint.Hash = attribute.value
				fingerprint.Text = attribute.extension
			case "candidate":
				if iceCandidates == nil {
					iceCandidates = []*ICECandidate{}
				}
				extensionSplit := strings.Split(attribute.extension, " ")
				iceCandidate := &ICECandidate{
					Component:  extensionSplit[0],
					Foundation: attribute.value,
					Ip:         extensionSplit[3],
					Port:       extensionSplit[4],
					Priority:   extensionSplit[2],
					Protocol:   extensionSplit[1],
					Type:       extensionSplit[6],
				}
				if iceCandidate.Type == "srflx" || iceCandidate.Type == "relay" {
					iceCandidate.RelAddr = extensionSplit[8]
					iceCandidate.RelPort = extensionSplit[10]
				}
				iceCandidates = append(iceCandidates, iceCandidate)
			}
			i++
		case "m":
			media := mediaParse(sdpValue)
			content := &Content{
				Creator: "initiator",
				Description: &RTPDescription{
					Media:        media.media,
					PayloadTypes: []*PayloadType{},
				},
				Transport: &ICEUDPTransport{
					FingerPrint: &FingerPrint{
						Hash: fingerprint.Hash,
						Text: fingerprint.Text,
					},
				},
			}
			for _, id := range media.ids {
				content.Description.PayloadTypes = append(content.Description.PayloadTypes, &PayloadType{
					Id: id,
				})
			}
			codecCounter := -1
			i += 2 // Skip c=IN IP4 0.0.0.0
			for i < len(sdplines) {
				finished := false
				curAttribute := attributeParse(sdplines[i][2:])
				switch curAttribute.name {
				case "setup":
					content.Transport.FingerPrint.Setup = curAttribute.value
				case "mid":
					content.Name = curAttribute.value
				case "ice-ufrag":
					content.Transport.UFrag = curAttribute.value
				case "ice-pwd":
					content.Transport.PWD = curAttribute.value
				case "rtpmap":
					codecCounter++
					codecData := strings.Split(curAttribute.extension, "/")
					content.Description.PayloadTypes[codecCounter].Name = codecData[0]
					content.Description.PayloadTypes[codecCounter].ClockRate = codecData[1]
					if len(codecData) > 2 {
						content.Description.PayloadTypes[codecCounter].Channels = codecData[2]
					}
				case "fmtp":
					content.Description.PayloadTypes[codecCounter].Parameter = []*struct {
						Name  string "xml:\"name,attr,omitempty\""
						Value string "xml:\"value,attr,omitempty\""
					}{}
					for _, parameter := range strings.Split(curAttribute.extension, ";") {
						parameterSplit := strings.Split(parameter, "=")
						content.Description.PayloadTypes[codecCounter].Parameter = append(content.Description.PayloadTypes[codecCounter].Parameter, &struct {
							Name  string "xml:\"name,attr,omitempty\""
							Value string "xml:\"value,attr,omitempty\""
						}{
							Name:  parameterSplit[0],
							Value: parameterSplit[1],
						})
					}
				case "rtcp-fb":
					if content.Description.PayloadTypes[codecCounter].RTCPFB == nil {
						content.Description.PayloadTypes[codecCounter].RTCPFB = []*struct {
							Type    string "xml:\"type,attr,omitempty\""
							SubType string "xml:\"subtype,attr,omitempty\""
						}{}
					}
					feedbackSplit := strings.Split(curAttribute.extension, " ")
					content.Description.PayloadTypes[codecCounter].RTCPFB = append(content.Description.PayloadTypes[codecCounter].RTCPFB, &struct {
						Type    string "xml:\"type,attr,omitempty\""
						SubType string "xml:\"subtype,attr,omitempty\""
					}{
						Type:    feedbackSplit[0],
						SubType: feedbackSplit[1],
					})
				case "ssrc":
					if content.Description.Source == nil {
						content.Description.Source = &struct {
							SSRC       string "xml:\"ssrc,attr,omitempty\""
							Parameters []struct {
								Name  string "xml:\"name,attr,omitempty\""
								Value string "xml:\"value,attr,omitempty\""
							} "xml:\"parameter,omitempty\""
						}{
							SSRC: curAttribute.value,
							Parameters: []struct {
								Name  string "xml:\"name,attr,omitempty\""
								Value string "xml:\"value,attr,omitempty\""
							}{},
						}
					}
					parameterSplit := strings.Split(curAttribute.extension, ":")
					content.Description.Source.Parameters = append(content.Description.Source.Parameters, struct {
						Name  string "xml:\"name,attr,omitempty\""
						Value string "xml:\"value,attr,omitempty\""
					}{
						Name:  parameterSplit[0],
						Value: parameterSplit[1],
					})
				case "send", "sendonly", "recv", "recvonly", "sendrecv", "inactive":
					finished = true
				}
				i++
				if finished {
					break
				}
			}
			j.Contents = append(j.Contents, content)
		default:
			i++
		}
	}
	if iceCandidates != nil {
		j.Contents[0].Transport.Candidates = append([]*ICECandidate{}, iceCandidates...)
	}

	return j
}

// Convert jingle message into SDP
func (j *Jingle) ToSDP() string {
	sdplines := []string{}
	sdplines = append(sdplines, []string{
		"v=0",
		"o=- 0 0 IN IP4 0.0.0.0",
		"s=-",
		"t=0 0",
	}...)

	if j.Contents[0].Transport.FingerPrint != nil {
		fingerprint := &attributeField{
			name:      "fingerprint",
			value:     j.Contents[0].Transport.FingerPrint.Hash,
			extension: j.Contents[0].Transport.FingerPrint.Text,
		}
		sdplines = append(sdplines, "a="+fingerprint.toString())
	}

	if j.Group != nil {
		contents := []string{}
		for _, content := range j.Group.Contents {
			contents = append(contents, content.Name)
		}
		group := &attributeField{
			name:      "group",
			value:     j.Group.Semantics,
			extension: strings.Join(contents, " "),
		}
		sdplines = append(sdplines, "a="+group.toString())
	}

	for idx, content := range j.Contents {
		media := &mediaField{
			media: content.Description.Media,
			ids:   []string{},
		}
		for _, payloadType := range content.Description.PayloadTypes {
			media.ids = append(media.ids, payloadType.Id)
		}
		sdplines = append(sdplines, "m="+media.toString())

		sdplines = append(sdplines, "c=IN IP4 0.0.0.0")

		if content.Transport.FingerPrint != nil {
			dtlsSetup := &attributeField{
				name:  "setup",
				value: content.Transport.FingerPrint.Setup,
			}
			sdplines = append(sdplines, "a="+dtlsSetup.toString())
		}

		mid := &attributeField{
			name:  "mid",
			value: content.Name,
		}
		sdplines = append(sdplines, "a="+mid.toString())

		iceUfrag := &attributeField{
			name:  "ice-ufrag",
			value: content.Transport.UFrag,
		}
		sdplines = append(sdplines, "a="+iceUfrag.toString())

		icePwd := &attributeField{
			name:  "ice-pwd",
			value: content.Transport.PWD,
		}
		sdplines = append(sdplines, "a="+icePwd.toString())

		sdplines = append(sdplines, []string{
			"a=rtcp-mux",
			"a=rtcp-rsize",
		}...)

		for _, payloadType := range content.Description.PayloadTypes {
			codecData := []string{
				payloadType.Name,
				payloadType.ClockRate,
			}
			if payloadType.Channels != "" {
				codecData = append(codecData, payloadType.Channels)
			}
			rtpmap := &attributeField{
				name:      "rtpmap",
				value:     payloadType.Id,
				extension: strings.Join(codecData, "/"),
			}
			sdplines = append(sdplines, "a="+rtpmap.toString())

			if payloadType.Parameter != nil {
				parameters := []string{}
				for _, parameter := range payloadType.Parameter {
					parameters = append(parameters, parameter.Name+"="+parameter.Value)
				}
				fmtp := &attributeField{
					name:      "fmtp",
					value:     payloadType.Id,
					extension: strings.Join(parameters, ";"),
				}
				sdplines = append(sdplines, "a="+fmtp.toString())
			}

			if payloadType.RTCPFB != nil {
				for _, rtcpfb := range payloadType.RTCPFB {
					rtcpfbVal := &attributeField{
						name:      "rtcp-fb",
						value:     payloadType.Id,
						extension: rtcpfb.Type + " " + rtcpfb.SubType,
					}
					sdplines = append(sdplines, "a="+rtcpfbVal.toString())
				}
			}
		}

		if content.Description.Source != nil {
			source := content.Description.Source
			msid := ""
			for _, parameter := range source.Parameters {
				ssrc := &attributeField{
					name:      "ssrc",
					value:     source.SSRC,
					extension: parameter.Name + ":" + parameter.Value,
				}
				if parameter.Name == "msid" {
					msid = parameter.Value
				}
				sdplines = append(sdplines, "a="+ssrc.toString())
			}
			msidVal := &attributeField{
				name:  "msid",
				value: msid,
			}
			sdplines = append(sdplines, "a="+msidVal.toString())
		}

		sdplines = append(sdplines, "a=sendrecv")

		if idx == 0 {
			for _, candidate := range content.Transport.Candidates {
				sdplines = append(sdplines, "a="+candidate.toSDP())
			}
			sdplines = append(sdplines, "a=end-of-candidates")
		}
	}
	sdplines = append(sdplines, "")
	return strings.Join(sdplines, "\r\n")
}
