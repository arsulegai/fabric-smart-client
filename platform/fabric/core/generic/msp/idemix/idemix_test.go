/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/
package idemix_test

import (
	"strings"
	"testing"
	"time"

	idemix2 "github.com/hyperledger-labs/fabric-smart-client/platform/fabric/core/generic/msp/idemix"
	sig2 "github.com/hyperledger-labs/fabric-smart-client/platform/view/core/sig"

	msp2 "github.com/hyperledger/fabric/msp"
	"github.com/stretchr/testify/assert"

	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/kvs"
	registry2 "github.com/hyperledger-labs/fabric-smart-client/platform/view/services/registry"
)

type fakeProv struct {
	typ  string
	path string
}

func (f *fakeProv) GetString(key string) string {
	return f.typ
}

func (f *fakeProv) GetDuration(key string) time.Duration {
	return time.Duration(0)
}

func (f *fakeProv) GetBool(key string) bool {
	return false
}

func (f *fakeProv) GetStringSlice(key string) []string {
	return nil
}

func (f *fakeProv) IsSet(key string) bool {
	return false
}

func (f *fakeProv) UnmarshalKey(key string, rawVal interface{}) error {
	*(rawVal.(*kvs.Opts)) = kvs.Opts{
		Path: f.path,
	}

	return nil
}

func (f *fakeProv) ConfigFileUsed() string {
	return ""
}

func (f *fakeProv) GetPath(key string) string {
	return ""
}

func (f *fakeProv) TranslatePath(path string) string {
	return ""
}

func TestProvider(t *testing.T) {
	registry := registry2.New()
	registry.RegisterService(&fakeProv{typ: "memory"})

	kvss, err := kvs.New("memory", "", registry)
	assert.NoError(t, err)
	assert.NoError(t, registry.RegisterService(kvss))
	sigService := sig2.NewSignService(registry, nil)
	assert.NoError(t, registry.RegisterService(sigService))

	config, err := msp2.GetLocalMspConfigWithType("./testdata/idemix", nil, "idemix", "idemix")
	assert.NoError(t, err)

	p, err := idemix2.NewProvider(config, registry)
	assert.NoError(t, err)
	assert.NotNil(t, p)
}

func TestIdentity(t *testing.T) {
	registry := registry2.New()
	registry.RegisterService(&fakeProv{typ: "memory"})

	kvss, err := kvs.New("memory", "", registry)
	assert.NoError(t, err)
	assert.NoError(t, registry.RegisterService(kvss))
	sigService := sig2.NewSignService(registry, nil)
	assert.NoError(t, registry.RegisterService(sigService))

	config, err := msp2.GetLocalMspConfigWithType("./testdata/idemix", nil, "idemix", "idemix")
	assert.NoError(t, err)

	p, err := idemix2.NewProvider(config, registry)
	assert.NoError(t, err)
	assert.NotNil(t, p)

	id, audit, err := p.Identity()
	assert.NoError(t, err)
	assert.NotNil(t, id)
	assert.NotNil(t, audit)
	info, err := p.Info(id, audit)
	assert.NoError(t, err)
	assert.True(t, strings.HasPrefix(info, "MSP.Idemix: [idemix]"))
	assert.True(t, strings.HasSuffix(info, "[idemix][idemixorg.example.com][ADMIN]"))

	auditInfo := &idemix2.AuditInfo{}
	assert.NoError(t, auditInfo.FromBytes(audit))
	assert.NoError(t, auditInfo.Match(id))

	signer, err := p.DeserializeSigner(id)
	assert.NoError(t, err)
	verifier, err := p.DeserializeVerifier(id)
	assert.NoError(t, err)

	sigma, err := signer.Sign([]byte("hello world!!!"))
	assert.NoError(t, err)
	assert.NoError(t, verifier.Verify([]byte("hello world!!!"), sigma))
}

func TestAudit(t *testing.T) {
	registry := registry2.New()
	registry.RegisterService(&fakeProv{typ: "memory"})

	kvss, err := kvs.New("memory", "", registry)
	assert.NoError(t, err)
	assert.NoError(t, registry.RegisterService(kvss))
	sigService := sig2.NewSignService(registry, nil)
	assert.NoError(t, registry.RegisterService(sigService))

	config, err := msp2.GetLocalMspConfigWithType("./testdata/idemix", nil, "idemix", "idemix")
	assert.NoError(t, err)
	p, err := idemix2.NewProvider(config, registry)
	assert.NoError(t, err)
	assert.NotNil(t, p)

	config, err = msp2.GetLocalMspConfigWithType("./testdata/idemix2", nil, "idemix", "idemix")
	assert.NoError(t, err)
	p2, err := idemix2.NewProvider(config, registry)
	assert.NoError(t, err)
	assert.NotNil(t, p2)

	id, audit, err := p.Identity()
	assert.NoError(t, err)
	assert.NotNil(t, id)
	assert.NotNil(t, audit)

	id2, audit2, err := p.Identity()
	assert.NoError(t, err)
	assert.NotNil(t, id2)
	assert.NotNil(t, audit2)

	auditInfo := &idemix2.AuditInfo{}
	assert.NoError(t, auditInfo.FromBytes(audit))
	assert.NoError(t, auditInfo.Match(id))
	assert.Error(t, auditInfo.Match(id2))

	auditInfo = &idemix2.AuditInfo{}
	assert.NoError(t, auditInfo.FromBytes(audit2))
	assert.NoError(t, auditInfo.Match(id2))
	assert.Error(t, auditInfo.Match(id))
}
