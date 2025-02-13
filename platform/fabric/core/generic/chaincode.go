/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/
package generic

import (
	"github.com/hyperledger-labs/fabric-smart-client/platform/fabric/api"
	"github.com/hyperledger-labs/fabric-smart-client/platform/fabric/core/generic/chaincode"
)

// Chaincode returns a chaincode handler for the passed chaincode name
func (c *channel) Chaincode(name string) api.Chaincode {
	return chaincode.NewChaincode(name, c.sp, c.network, c)
}
