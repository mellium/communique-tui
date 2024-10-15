// Copyright 2022 The Mellium Contributors.
// Use of this source code is governed by the BSD 2-clause
// license that can be found in the LICENSE file.

package main

import (
	"context"
	"encoding/xml"
	"log"

	"mellium.im/communique/internal/client"
	"mellium.im/communique/internal/ui"
	"mellium.im/xmlstream"
	"mellium.im/xmpp/commands"
	"mellium.im/xmpp/form"
	"mellium.im/xmpp/oob"
)

func showCmd(pane *ui.UI, client *client.Client, resp commands.Response, payload xmlstream.TokenReadCloser, debug *log.Logger) error {
	var (
		actions  commands.Actions
		note     commands.Note
		formData *form.Data
	)
	err := func() (err error) {
		defer func() {
			e := payload.Close()
			if err == nil && e != nil {
				err = e
			}
		}()
		iter := xmlstream.NewIter(payload)
		for iter.Next() {
			start, inner := iter.Current()
			if start == nil {
				continue
			}

			d := xml.NewTokenDecoder(xmlstream.Wrap(inner, *start))
			// Pop the start element to put the decoder in the correct state.
			_, err := d.Token()
			if err != nil {
				return err
			}
			switch {
			case start.Name.Space == commands.NS && start.Name.Local == "note":
				if note.Value != "" || formData != nil {
					continue
				}
				err := d.DecodeElement(&note, start)
				if err != nil {
					return err
				}
			case start.Name.Space == oob.NS:
				if note.Value != "" || formData != nil {
					continue
				}
				var oobURL oob.Data
				err := d.DecodeElement(&oobURL, start)
				if err != nil {
					return err
				}
				note = commands.Note{
					Value: oobURL.Desc + "\n\n" + oobURL.URL,
				}
			case start.Name.Space == form.NS:
				if note.Value != "" || formData != nil {
					continue
				}
				formData = &form.Data{}
				err := d.DecodeElement(formData, start)
				if err != nil {
					return err
				}
			case start.Name.Space == commands.NS && start.Name.Local == "actions":
				// Just decode the actions, they will be displayed at the end.
				err := d.DecodeElement(&actions, start)
				if err != nil {
					return err
				}
			}
		}
		return iter.Err()
	}()
	if err != nil {
		return err
	}

	p := pane.Printer()
	var (
		prevBtn     = p.Sprintf("Prev")
		nextBtn     = p.Sprintf("Next")
		completeBtn = p.Sprintf("Complete")
		cancelBtn   = p.Sprintf("Cancel")
	)
	var buttons []string
	if actions&commands.Prev == commands.Prev {
		buttons = append(buttons, prevBtn)
	}
	if actions&commands.Next == commands.Next {
		buttons = append(buttons, nextBtn)
	}
	if actions == 0 || actions&commands.Complete == commands.Complete {
		buttons = append(buttons, completeBtn)
	}
	buttons = append(buttons, cancelBtn)

	onDone := func(label string) {
		pane.SelectRoster()

		var nextCmd commands.Command
		switch label {
		case prevBtn:
			nextCmd = resp.Prev()
		case nextBtn:
			nextCmd = resp.Next()
		case completeBtn:
			nextCmd = resp.Complete()
		default:
			if resp.Status != "completed" && resp.Status != "canceled" {
				// If for some reason we can't cancel the command we don't need to wait
				// on it to complete to close the dialog.
				go func() {
					ctx, cancel := context.WithTimeout(context.Background(), client.Timeout())
					defer cancel()
					_, trc, err := resp.Cancel().Execute(ctx, nil, client.Session)
					if err != nil {
						debug.Print(p.Sprintf("error canceling command session: %v", err))
					}
					if trc != nil {
						err = trc.Close()
						if err != nil {
							debug.Print(p.Sprintf("error closing cancel command payload: %v", err))
						}
					}
				}()
			}
			return
		}

		ctx, cancel := context.WithTimeout(context.Background(), client.Timeout())
		defer cancel()
		var payload xml.TokenReader
		if formData != nil {
			payload, _ = formData.Submit()
		}
		if resp.Status != "completed" && resp.Status != "canceled" {
			resp, trc, err := nextCmd.Execute(ctx, payload, client.Session)
			if err != nil {
				debug.Print(p.Sprintf("error closing command session: %v", err))
			}
			go func() {
				err = showCmd(pane, client, resp, trc, debug)
				if err != nil {
					debug.Print(p.Sprintf("error showing next command: %v", err))
				}
			}()
		}
	}

	switch {
	case formData != nil:
		pane.ShowForm(formData, buttons, onDone)
	case note.Value != "":
		pane.ShowNote(note, buttons, onDone)
	}
	return nil
}
