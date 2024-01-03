package main

import (
	"mellium.im/communique/internal/client"
	"mellium.im/communique/internal/client/event"
)

func newTestHandler(c *client.Client) func(interface{}) {
	return func(ev interface{}) {
		switch e := ev.(type) {
		case event.Receipt:
			mc, ok := messageIds[string(e)]
			if ok {
				mc <- true
			}
		}
	}
}
