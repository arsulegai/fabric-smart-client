/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/
package pingpong

import (
	"fmt"
	"time"

	"github.com/pkg/errors"

	"github.com/hyperledger-labs/fabric-smart-client/platform/generic"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/assert"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/view"
)

type Initiator struct{}

func (p *Initiator) Call(context view.Context) (interface{}, error) {
	// Retrieve responder identity
	responder := generic.GetIdentityProvider(context).Identity("responder")

	// Open a session to the responder
	session, err := context.GetSession(context.Initiator(), responder)
	assert.NoError(err) // Send a ping
	err = session.Send([]byte("ping"))
	assert.NoError(err) // Wait for the pong
	ch := session.Receive()
	select {
	case msg := <-ch:
		if msg.Status == view.ERROR {
			return nil, errors.New(string(msg.Payload))
		}
		m := string(msg.Payload)
		if m != "pong" {
			return nil, fmt.Errorf("exptectd pong, got %s", m)
		}
	case <-time.After(1 * time.Minute):
		return nil, errors.New("responder didn't pong in time")
	}

	// Return
	return "OK", nil
}
