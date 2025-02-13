/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/
package views

import (
	"encoding/json"

	"github.com/hyperledger-labs/fabric-smart-client/platform/fabric/services/state"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/assert"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/view"
)

type AgreeToSell struct {
	Agreement *AgreementToSell

	Approver view.Identity
}

type AgreeToSellView struct {
	*AgreeToSell
}

func (a *AgreeToSellView) Call(context view.Context) (interface{}, error) {
	// Prepare transaction
	tx, err := state.NewTransaction(context)
	assert.NoError(err, "failed creating transaction")
	tx.SetNamespace("asset_transfer")
	assert.NoError(tx.AddCommand("agreeToSell", context.Me()), "failed adding issue command")

	a.Agreement.Owner = context.Me()
	assert.NoError(tx.AddOutput(a.Agreement, state.WithHashHiding()), "failed adding output")

	_, err = context.RunView(state.NewCollectEndorsementsView(tx, context.Me()))
	assert.NoError(err, "failed collecting endorsement")

	_, err = context.RunView(state.NewCollectApprovesView(tx, a.Approver))
	assert.NoError(err, "failed collecting approves")

	// Send to the ordering service and wait for confirmation
	_, err = context.RunView(state.NewOrderingView(tx))
	assert.NoError(err, "failed asking ordering")

	return tx.ID(), nil
}

type AgreeToSellViewFactory struct{}

func (a *AgreeToSellViewFactory) NewView(in []byte) (view.View, error) {
	f := &AgreeToSellView{AgreeToSell: &AgreeToSell{}}
	err := json.Unmarshal(in, f.AgreeToSell)
	assert.NoError(err, "failed unmarshalling input")
	return f, nil
}

type AgreeToBuy struct {
	Agreement *AgreementToBuy

	Approver view.Identity
}

type AgreeToBuyView struct {
	*AgreeToBuy
}

func (a *AgreeToBuyView) Call(context view.Context) (interface{}, error) {
	// Prepare transaction
	tx, err := state.NewTransaction(context)
	assert.NoError(err, "failed creating transaction")
	tx.SetNamespace("asset_transfer")
	assert.NoError(tx.AddCommand("agreeToBuy", context.Me()), "failed adding issue command")

	a.Agreement.Owner = context.Me()
	assert.NoError(tx.AddOutput(a.Agreement, state.WithHashHiding()), "failed adding output")

	_, err = context.RunView(state.NewCollectEndorsementsView(tx, context.Me()))
	assert.NoError(err, "failed collecting endorsement")

	_, err = context.RunView(state.NewCollectApprovesView(tx, a.Approver))
	assert.NoError(err, "failed collecting approves")

	// Send to the ordering service and wait for confirmation
	_, err = context.RunView(state.NewOrderingView(tx))
	assert.NoError(err, "failed asking ordering")

	return tx.ID(), nil
}

type AgreeToBuyViewFactory struct{}

func (a *AgreeToBuyViewFactory) NewView(in []byte) (view.View, error) {
	f := &AgreeToBuyView{AgreeToBuy: &AgreeToBuy{}}
	err := json.Unmarshal(in, f.AgreeToBuy)
	assert.NoError(err, "failed unmarshalling input")
	return f, nil
}
