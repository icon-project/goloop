/*
 * Copyright 2023 ICON Foundation
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 *
 */

package service

import (
	"bytes"
	"encoding/json"
	"fmt"
	"reflect"

	"github.com/icon-project/goloop/chain/base"
	"github.com/icon-project/goloop/common/db"
	"github.com/icon-project/goloop/common/log"
	"github.com/icon-project/goloop/common/trie"
	"github.com/icon-project/goloop/common/trie/trie_manager"
	"github.com/icon-project/goloop/module"
	"github.com/icon-project/goloop/service/state"
	"github.com/icon-project/goloop/service/txresult"
)

const (
	DNExtension = "extension"
)
type ObjectDetailHandler func(name string, key []byte, exp, real trie.Object)

type DiffContext interface {
	Database() db.Database
	Logger() log.Logger

	ShowObjectMPTDiff(name string, dbase db.Database, t reflect.Type, e, r []byte, handler ObjectDetailHandler) error
}

type PlatformWithShowDiff interface {
	ShowDiff(ctx DiffContext, name string, e, r []byte) error
}

type diffContext struct {
	plt   base.Platform
	dbase db.Database
	log   log.Logger
}

func (c *diffContext) Database() db.Database {
	return c.dbase
}

func (c *diffContext) Logger() log.Logger {
	return c.log
}

func (c *diffContext) GetObjectDiffHandlerFor(name string, handler ObjectDetailHandler) trie_manager.ObjectDifferenceHandler {
	return func(op int, key []byte, exp, real trie.Object) {
		switch op {
		case -1:
			c.log.Errorf("%s [-] key=%#x value=%+v\n", name, key, exp)
		case 0:
			if exp.Equal(real) {
				c.log.Errorf("%s [=] key=%#x exp=<%#x> real=<%#x>\n", name, key, exp.Bytes(), real.Bytes())
			} else {
				c.log.Errorf("%s [=] key=%#x exp=%+v real=%+v\n", name, key, exp, real)
			}
			if handler != nil {
				handler(name, key, exp, real)
			}
		case 1:
			c.log.Errorf("%s [+] key=%#x value=%+v\n", name, key, real)
		}
	}
}

func (c *diffContext) GetBytesDiffHandlerFor(name string) trie_manager.BytesDifferenceHandler {
	return func(op int, key []byte, exp, real []byte) {
		switch op {
		case -1:
			c.log.Errorf("%s [-] key=%#x value=<%#x>\n", name, key, exp)
		case 0:
			c.log.Errorf("%s [=] key=%#x exp=<%#x> real=<%#x>\n", name, key, exp, real)
		case 1:
			c.log.Errorf("%s [+] key=%#x value=<%#x>\n", name, key, real)
		}
	}
}

func (c *diffContext) ShowObjectMPTDiff(name string, dbase db.Database, t reflect.Type, e, r []byte, handler ObjectDetailHandler) error {
	et := trie_manager.NewImmutableForObject(dbase, e, t)
	rt := trie_manager.NewImmutableForObject(dbase, r, t)
	return trie_manager.CompareImmutableForObject(et, rt, c.GetObjectDiffHandlerFor(name, handler))
}

func (c *diffContext) AccountDetailHandler() ObjectDetailHandler {
	type Storer interface {
		Store() trie.Immutable
	}
	return func (name string, key []byte, exp, real trie.Object) {
		var eStore, rStore trie.Immutable
		if eASS, ok := exp.(Storer); ok {
			eStore = eASS.Store()
		}
		if rASS, ok := real.(Storer); ok {
			rStore = rASS.Store()
		}
		if eStore == rStore {
			return
		}
		if eStore == nil {
			c.log.Errorf("%s [+] key=%#x real=%+v", name+".store", key, rStore)
			return
		} else if rStore == nil {
			c.log.Errorf("%s [-] key=%#x exp=%+v", name+".store", key, eStore)
			return
		}
		accountHash := fmt.Sprintf("%#x", key)
		err := trie_manager.CompareImmutable(eStore, rStore,
			c.GetBytesDiffHandlerFor(accountHash))
		if err != nil {
			c.log.Errorf("%s fail to compare store", name)
		}
	}
}

func JSONMarshalIndent(obj interface{}) ([]byte, error) {
	type ToJSONer interface {
		ToJSON(version module.JSONVersion) (interface{}, error)
	}
	if jsoner, ok := obj.(ToJSONer); ok {
		if jso, err := jsoner.ToJSON(module.JSONVersionLast); err == nil {
			obj = jso
		} else {
			log.Warnf("Failure in ToJSON err=%+v", err)
		}
	}
	return json.MarshalIndent(obj, "", "  ")
}


func (c *diffContext) showReceiptDiff(name string, e, r []byte) error {
	el := txresult.NewReceiptListFromHash(c.dbase, e)
	rl := txresult.NewReceiptListFromHash(c.dbase, r)
	idx := 0
	for expect, result := el.Iterator(), rl.Iterator(); expect.Has() && result.Has(); _, _, idx = expect.Next(), result.Next(), idx+1 {
		rct1, _ := expect.Get()
		rct2, _ := result.Get()
		if err := rct1.Check(rct2); err != nil {
			rct1js, _ := JSONMarshalIndent(rct1)
			rct2js, _ := JSONMarshalIndent(rct2)
			c.log.Errorf("Expected %s Receipt[%d]:%s", name, idx, rct1js)
			c.log.Errorf("Returned %s Receipt[%d]:%s", name, idx, rct2js)
		}
	}
	return nil
}

func (c *diffContext) showResultDiff(e, r *transitionResult) error {
	if !bytes.Equal(e.StateHash, r.StateHash) {
		if err := c.ShowObjectMPTDiff("world", c.dbase, state.AccountType,
			e.StateHash, r.StateHash, c.AccountDetailHandler()) ; err != nil {
			return err
		}
	}
	if !bytes.Equal(e.PatchReceiptHash, r.PatchReceiptHash) {
		if err := c.showReceiptDiff("Patch", e.PatchReceiptHash, r.PatchReceiptHash) ; err != nil {
			return err
		}
	}
	if !bytes.Equal(e.NormalReceiptHash, r.NormalReceiptHash) {
		if err := c.showReceiptDiff("Normal", e.NormalReceiptHash, r.NormalReceiptHash) ; err != nil {
			return err
		}
	}
	if !bytes.Equal(e.ExtensionData, r.ExtensionData) {
		c.log.Errorf("ExtensionData [=] e=<%#x> r=<%#x>", e.ExtensionData, r.ExtensionData)
		if plt, ok := c.plt.(PlatformWithShowDiff); ok {
			if err := plt.ShowDiff(c, DNExtension, e.ExtensionData, r.ExtensionData); err != nil {
				return err
			}
		}
	}
	if !bytes.Equal(e.BTPData, r.BTPData) {
		c.log.Errorf("BTPData [=] e=<%#x> r=<%#x>", e.BTPData, r.BTPData)
	}
	return nil
}

func ShowResultDiff(dbase db.Database, plt base.Platform, logger log.Logger, exp, real []byte) error {
	if bytes.Equal(exp, real) {
		return nil
	}
	eResult, err := newTransitionResultFromBytes(exp)
	if err != nil {
		return err
	}
	rResult, err := newTransitionResultFromBytes(real)
	if err != nil {
		return err
	}
	c := &diffContext{
		plt:   plt,
		dbase: dbase,
		log:   logger,
	}
	return c.showResultDiff(eResult, rResult)
}
