/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/
package api

import (
	"reflect"

	"github.com/hyperledger-labs/fabric-smart-client/platform/view/view"
)

//go:generate counterfeiter -o mock/id_provider.go -fake-name IdentityProvider . IdentityProvider

// IdentityProvider models the identity provider
type IdentityProvider interface {
	// DefaultIdentity returns the default identity known by this provider
	DefaultIdentity() view.Identity
}

func GetIdentityProvider(sp ServiceProvider) IdentityProvider {
	s, err := sp.GetService(reflect.TypeOf((*IdentityProvider)(nil)))
	if err != nil {
		panic(err)
	}
	return s.(IdentityProvider)
}
