/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/
package kvs

import (
	"encoding/json"
	"path/filepath"
	"sync"

	"github.com/pkg/errors"

	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/flogging"

	"github.com/hyperledger-labs/fabric-smart-client/platform/view"
	view2 "github.com/hyperledger-labs/fabric-smart-client/platform/view"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/db"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/db/driver"
)

var logger = flogging.MustGetLogger("view-sdk.kvs")

type KVS struct {
	namespace string
	store     driver.Persistence

	putMutex sync.Mutex
}

type Opts struct {
	Path string
}

func New(driverName, namespace string, sp view.ServiceProvider) (*KVS, error) {
	opts := &Opts{}
	err := view2.GetConfigService(sp).UnmarshalKey("fsc.kvs.persistence.opts", opts)
	if err != nil {
		return nil, errors.Wrapf(err, "failed getting opts for vault")
	}
	path := filepath.Join(opts.Path, namespace)

	persistence, err := db.Open(driverName, path)
	if err != nil {
		return nil, errors.WithMessagef(err, "no driver found for [%s]", driverName)
	}

	return &KVS{namespace: namespace, store: persistence}, nil
}

func (o *KVS) Exists(id string) bool {
	raw, err := o.store.GetState(o.namespace, id)
	if err != nil {
		logger.Errorf("failed getting state [%s,%s]", o.namespace, id)
		return false
	}
	logger.Debugf("state [%s,%s] exists [%v]", o.namespace, id, len(raw) != 0)
	return len(raw) != 0
}

func (o *KVS) Put(id string, state interface{}) error {
	logger.Debugf("put state [%s,%s]", o.namespace, id)

	o.putMutex.Lock()
	defer o.putMutex.Unlock()

	raw, err := json.Marshal(state)
	if err != nil {
		return errors.Wrapf(err, "cannot marshal state with id [%s]", id)
	}

	err = o.store.BeginUpdate()
	if err != nil {
		return errors.WithMessagef(err, "begin update for id [%s] failed", id)
	}

	err = o.store.SetState(o.namespace, id, raw)
	if err != nil {
		if err1 := o.store.Discard(); err1 != nil {
			logger.Errorf("got error %s; discarding caused %s", err.Error(), err1.Error())
		}

		return errors.Errorf("failed to commit value for id [%s]", id)
	}

	err = o.store.Commit()
	if err != nil {
		return errors.WithMessagef(err, "committing value for id [%s] failed", id)
	}

	return nil
}

func (o *KVS) Get(id string, state interface{}) error {
	raw, err := o.store.GetState(o.namespace, id)
	if err != nil {
		logger.Errorf("failed retrieving state [%s,%s]", o.namespace, id)
		return errors.Errorf("failed retrieving state [%s,%s]", o.namespace, id)
	}
	if err := json.Unmarshal(raw, state); err != nil {
		logger.Errorf("failed retrieving state [%s,%s], cannot unmarshal state, error [%s]", o.namespace, id, err)
		return errors.Wrapf(err, "failed retrieving state [%s,%s], cannot unmarshal state", o.namespace, id)
	}
	logger.Debugf("got state [%s,%s] successfully", o.namespace, id)
	return nil
}

func (o *KVS) GetByPartialCompositeID(prefix string, attrs []string) (*iteratorConverter, error) {
	partialCompositeKey, err := CreateCompositeKey(prefix, attrs)
	if err != nil {
		return nil, errors.Errorf("failed building composite key [%s]", err)
	}

	startKey := partialCompositeKey
	endKey := partialCompositeKey + string(maxUnicodeRuneValue)

	itr, err := o.store.GetStateRangeScanIterator(o.namespace, startKey, endKey)
	if err != nil {
		return nil, errors.Errorf("store access failure for GetStateRangeScanIterator [%s], ns [%s] range [%s,%s]", err, o.namespace, startKey, endKey)
	}

	return &iteratorConverter{ri: itr}, nil
}

func (o *KVS) Stop() {
	if err := o.store.Close(); err != nil {
		logger.Errorf("failed stopping kvs [%s]", err)
	}
}

type iteratorConverter struct {
	ri   driver.ResultsIterator
	next *driver.Read
}

func (i *iteratorConverter) HasNext() bool {
	var err error
	i.next, err = i.ri.Next()
	if err != nil || i.next == nil {
		return false
	}
	return true
}

func (i *iteratorConverter) Close() error {
	i.ri.Close()
	return nil
}

func (i *iteratorConverter) Next(state interface{}) error {
	return json.Unmarshal(i.next.Raw, state)
}

func GetService(ctx view2.ServiceProvider) *KVS {
	s, err := ctx.GetService(&KVS{})
	if err != nil {
		panic(err)
	}
	return s.(*KVS)
}
