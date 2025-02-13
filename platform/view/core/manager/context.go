/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/
package manager

import (
	"context"
	"runtime/debug"
	"sync"

	"github.com/pkg/errors"

	view2 "github.com/hyperledger-labs/fabric-smart-client/platform/view"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/api"
	sig2 "github.com/hyperledger-labs/fabric-smart-client/platform/view/core/sig"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/view"
)

type ctx struct {
	context        context.Context
	sp             api.ServiceProvider
	id             string
	session        view.Session
	initiator      view.View
	me             view.Identity
	caller         view.Identity
	resolver       api.EndpointService
	sessionFactory SessionFactory

	sessionsLock sync.RWMutex
	sessions     map[string]view.Session
}

func NewContextForInitiator(context context.Context, sp api.ServiceProvider, sessionFactory SessionFactory, resolver api.EndpointService, party view.Identity, initiator view.View) (*ctx, error) {
	ctx, err := NewContext(context, sp, GenerateUUID(), sessionFactory, resolver, party, nil, nil)
	if err != nil {
		return nil, err
	}
	ctx.initiator = initiator

	return ctx, nil
}

func NewContext(context context.Context, sp api.ServiceProvider, contextID string, sessionFactory SessionFactory, resolver api.EndpointService, party view.Identity, session view.Session, caller view.Identity) (*ctx, error) {
	ctx := &ctx{
		context:        context,
		id:             contextID,
		resolver:       resolver,
		sessionFactory: sessionFactory,
		session:        session,
		me:             party,
		sessions:       map[string]view.Session{},
		caller:         caller,
		sp:             sp,
	}
	return ctx, nil
}

func (ctx *ctx) ID() string {
	return ctx.id
}

func (ctx *ctx) Initiator() view.View {
	return ctx.initiator
}

func (ctx *ctx) RunView(view view.View) (res interface{}, err error) {
	wContext := &wrappedContext{ctx: ctx}
	defer func() {
		if r := recover(); r != nil {
			wContext.cleanup()
			res = nil

			logger.Debugf("caught panic while running view with [%v][%s]", r, debug.Stack())

			switch e := r.(type) {
			case error:
				err = errors.WithMessage(e, "caught panic")
			case string:
				err = errors.Errorf(e)
			default:
				err = errors.Errorf("caught panic [%v]", e)
			}
		}
	}()
	res, err = view.Call(wContext)
	if err != nil {
		wContext.cleanup()
		return nil, err
	}
	return res, err
}

func (ctx *ctx) Me() view.Identity {
	return ctx.me
}

// TODO: remove this
func (ctx *ctx) Identity(ref string) (view.Identity, error) {
	return api.GetEndpointService(ctx.sp).GetIdentity(ref, nil)
}

func (ctx *ctx) IsMe(id view.Identity) bool {
	_, err := sig2.GetSigner(ctx, id)
	return err == nil
}

func (ctx *ctx) Caller() view.Identity {
	return ctx.caller
}

func (ctx *ctx) GetSession(f view.View, party view.Identity) (view.Session, error) {
	// TODO: we need a mechanism to close all the sessions opened in this ctx,
	// when the ctx goes out of scope
	ctx.sessionsLock.Lock()
	defer ctx.sessionsLock.Unlock()

	var err error
	id := party
	s, ok := ctx.sessions[id.UniqueID()]
	if !ok {
		// TODO: do we need a recursion here?
		id, _, _, err = view2.GetEndpointService(ctx).Resolve(party)
		if err == nil {
			s, ok = ctx.sessions[id.UniqueID()]
		}
	}

	if ok && s.Info().Closed {
		// Remove this session cause it is closed
		delete(ctx.sessions, id.UniqueID())
		ok = false
	}

	if !ok {
		logger.Debugf("[%s] Creating new session [to:%s]", ctx.me, party)
		s, err = ctx.newSession(f, ctx.id, party)
		if err != nil {
			return nil, err
		}
		ctx.sessions[party.UniqueID()] = s
	} else {
		logger.Debugf("[%s] Reusing session [to:%s]", ctx.me, party)
	}
	return s, nil
}

func (ctx *ctx) GetSessionByID(id string, party view.Identity) (view.Session, error) {
	ctx.sessionsLock.Lock()
	defer ctx.sessionsLock.Unlock()

	var err error
	key := id + "." + party.UniqueID()
	s, ok := ctx.sessions[key]
	if !ok {
		logger.Debugf("[%s] Creating new session with given id [id:%s][to:%s]", ctx.me, id, party)
		s, err = ctx.newSessionByID(id, ctx.id, party)
		if err != nil {
			return nil, err
		}
		ctx.sessions[key] = s
	} else {
		logger.Debugf("[%s] Reusing session with given id [id:%s][to:%s]", id, ctx.me, party)
	}
	return s, nil
}

func (ctx *ctx) Session() view.Session {
	if ctx.session == nil {
		logger.Debugf("[%s] No default current Session", ctx.me)
		return nil
	}
	logger.Debugf("[%s] Current Session [%s]", ctx.me, ctx.session.Info())
	return ctx.session
}

func (ctx *ctx) GetService(v interface{}) (interface{}, error) {
	return ctx.sp.GetService(v)
}

func (ctx *ctx) OnError(callback func()) {
	panic("this cannot be invoked here")
}

func (ctx *ctx) Context() context.Context {
	return ctx.context
}

func (ctx *ctx) newSession(view view.View, contextID string, party view.Identity) (view.Session, error) {
	_, endpoints, pkid, err := ctx.resolver.Resolve(party)
	if err != nil {
		return nil, err
	}
	return ctx.sessionFactory.NewSession(getIdentifier(view), contextID, endpoints[api.P2PPort], pkid)
}

func (ctx *ctx) newSessionByID(sessionID, contextID string, party view.Identity) (view.Session, error) {
	_, endpoints, pkid, err := ctx.resolver.Resolve(party)
	if err != nil {
		return nil, err
	}
	return ctx.sessionFactory.NewSessionWithID(sessionID, contextID, endpoints[api.P2PPort], pkid, nil, nil)
}
