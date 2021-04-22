/*
 * Copyright 2021 ICON Foundation
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
 */

package main

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"strconv"
	"strings"
	"text/scanner"

	"github.com/icon-project/goloop/common"
	"github.com/icon-project/goloop/common/codec"
	"github.com/icon-project/goloop/common/containerdb"
	"github.com/icon-project/goloop/common/db"
	"github.com/icon-project/goloop/common/errors"
	"github.com/icon-project/goloop/common/intconv"
	"github.com/icon-project/goloop/common/log"
	"github.com/icon-project/goloop/common/trie"
	"github.com/icon-project/goloop/common/trie/trie_manager"
	"github.com/icon-project/goloop/icon/iiss/icobject"
	"github.com/icon-project/goloop/icon/iiss/icreward"
	"github.com/icon-project/goloop/icon/iiss/icstage"
	"github.com/icon-project/goloop/icon/iiss/icstate"
	"github.com/icon-project/goloop/module"
	"github.com/icon-project/goloop/service/scoredb"
	"github.com/icon-project/goloop/service/state"
	"github.com/icon-project/goloop/service/txresult"
)

func parseParams(p string) ([]string, error) {
	var params []string
	s := new(scanner.Scanner)
	s.Init(bytes.NewBufferString(p))
	s.Mode = scanner.ScanIdents | scanner.ScanStrings | scanner.ScanInts
	for {
		switch value := s.Scan(); value {
		case scanner.EOF:
			return params, nil
		case scanner.Ident, scanner.Int:
			params = append(params, s.TokenText())
		case scanner.String, scanner.RawString:
			token := s.TokenText()
			params = append(params, token)
		case '-':
			if s.Scan() == scanner.Int {
				params = append(params, "-"+s.TokenText())
			} else {
				return nil, errors.IllegalArgumentError.Errorf("InvalidTokenAfterMinus")
			}
		case '*':
			params = append(params, "*")
		case '.':
		default:
			return nil, errors.IllegalArgumentError.Errorf(
				"Unknown character=%c", value)
		}
	}
}

func toKeys(params []string) []interface{} {
	var keys []interface{}
	for _, p := range params {
		if p[0] == '"' {
			if s, err := strconv.Unquote(p); err == nil {
				keys = append(keys, s)
			} else {
				keys = append(keys, p)
			}
			continue
		}
		if strings.HasPrefix(p, "hx") || strings.HasPrefix(p, "cx") {
			if addr, err := common.NewAddressFromString(p); err == nil {
				keys = append(keys, addr)
				continue
			}
		}
		if strings.HasPrefix(p, "bx") {
			if bs, err := hex.DecodeString(p[2:]); err == nil {
				keys = append(keys, bs)
				continue
			}
		}
		if v, err := strconv.ParseInt(p, 0, 64); err == nil {
			keys = append(keys, v)
		} else {
			keys = append(keys, p)
		}
	}
	return keys
}

func showValue(value containerdb.Value, ts string) {
	if value == nil {
		fmt.Println("nil")
		return
	}
	ts = strings.ToLower(ts)
	switch ts {
	case "int":
		if v := value.BigInt(); v != nil {
			fmt.Printf("%d\n", v)
		} else {
			fmt.Println("0")
		}
	case "hexint":
		if v := value.BigInt(); v != nil {
			fmt.Println(intconv.FormatBigInt(v))
		} else {
			fmt.Println("0x0")
		}
	case "bool":
		fmt.Printf("%v\n", value.Bool())
	case "str", "string":
		fmt.Printf("%q\n", value.String())
	case "addr", "address":
		if addr := value.Address(); addr != nil {
			fmt.Println(addr.String())
		} else {
			fmt.Println("nil")
		}
	case "bytes":
		fmt.Printf("%#x\n", value.Bytes())
	case "obj", "object":
		fmt.Printf("%+v\n", value.Object())
	default:
		log.Warnf("Unknown type=%s bytes=%#x", ts, value.Bytes())
	}
}

func showArray(array *containerdb.ArrayDB, params []string, ts string) error {
	pLen := len(params)
	if pLen == 0 {
		fmt.Printf("%d\n", array.Size())
		return nil
	}
	if pLen != 1 {
		return errors.IllegalArgumentError.Errorf("InvalidParameterForArray(params=%+v)", params)
	}
	param := params[0]
	switch param {
	case "size":
		fmt.Printf("%d\n", array.Size())
		return nil
	case "all", "*":
		size := array.Size()
		for i := 0; i < size; i++ {
			showValue(array.Get(i), ts)
		}
		return nil
	default:
		if idx, err := strconv.ParseInt(param, 0, 64); err != nil {
			return errors.IllegalArgumentError.Wrapf(err, "InvalidArrayParameter(param=%s)", param)
		} else {
			showValue(array.Get(int(idx)), ts)
			return nil
		}
	}
}

func showContainerData(store interface{}, params []string) error {
	pLen := len(params)
	if pLen < 3 {
		return errors.IllegalArgumentError.Errorf("InvalidParameterCount(params=%+v)", params)
	}
	ts := params[pLen-1]
	ct := params[0]
	params = params[1:pLen-1]
	pLen = len(params)

	handleDict := func(hashType containerdb.KeyBuilderType) error {
		keys := toKeys(params)
		kb := containerdb.ToKey(hashType, keys...)
		value := containerdb.NewVarDB(store, kb)
		showValue(value, ts)
		return nil
	}

	handleArray := func(hashType containerdb.KeyBuilderType) error {
		if ts == "size" {
			keys := toKeys(params)
			kb := containerdb.ToKey(hashType).Append(keys...)
			array := containerdb.NewArrayDB(store, kb)
			return showArray(array, []string{}, ts)
		} else {
			keys := toKeys(params[:pLen-1])
			param := params[pLen-1]
			kb := containerdb.ToKey(hashType, keys...)
			array := containerdb.NewArrayDB(store, kb)
			return showArray(array, []string{param}, ts)
		}
	}

	switch ct {
	case "var":
		keys := toKeys(params)
		kb := containerdb.ToKey(containerdb.HashBuilder, scoredb.VarDBPrefix).Append(keys...)
		showValue(containerdb.NewVarDB(store, kb), ts)
		return nil
	case "array":
		if ts == "size" {
			keys := toKeys(params)
			kb := containerdb.ToKey(containerdb.HashBuilder, scoredb.ArrayDBPrefix).Append(keys...)
			array := containerdb.NewArrayDB(store, kb)
			return showArray(array, []string{}, ts)
		} else {
			keys := toKeys(params[:pLen-1])
			param := params[pLen-1]
			kb := containerdb.ToKey(containerdb.HashBuilder, scoredb.ArrayDBPrefix).Append(keys...)
			array := containerdb.NewArrayDB(store, kb)
			return showArray(array, []string{param}, ts)
		}
	case "dict":
		keys := toKeys(params)
		kb := containerdb.ToKey(containerdb.HashBuilder, scoredb.DictDBPrefix)
		dict := containerdb.NewDictDB(store, len(keys), kb)
		value := dict.Get(keys...)
		showValue(value, ts)
		return nil
	case "hash", "hash_var", "hash_dict":
		return handleDict(containerdb.HashBuilder)
	case "hash_array":
		return handleArray(containerdb.HashBuilder)
	case "raw", "raw_var", "raw_dict":
		return handleDict(containerdb.RawBuilder)
	case "raw_array":
		return handleArray(containerdb.RawBuilder)
	case "rlp", "rlp_var", "rlp_dict":
		return handleDict(containerdb.RLPBuilder)
	case "rlp_array":
		return handleArray(containerdb.RLPBuilder)
	case "ph", "ph_var", "ph_dict":
		return handleDict(containerdb.PrefixedHashBuilder)
	case "ph_array":
		return handleArray(containerdb.PrefixedHashBuilder)
	default:
		return errors.IllegalArgumentError.Errorf("InvalidContentType(%s)", ct)
	}
}


func showAccount(addr module.Address, ass state.AccountSnapshot, params []string) error {
	if len(params) == 0 {
		fmt.Printf("Account[%s]\n", addr.String())
		fmt.Printf("- Balance : %#d\n", ass.GetBalance())
		if ass.IsContract() {
			fmt.Printf("- Owner   : %s\n", ass.ContractOwner())
			fmt.Printf("- CodeHash: %#x\n", ass.Contract().CodeHash())
		}
		return nil
	} else {
		token := params[0]
		switch token {
		case "var", "array", "dict":
			store := containerdb.NewBytesStoreStateWithSnapshot(ass)
			return showContainerData(store, params)
		case "api":
			api, err := ass.APIInfo()
			if err != nil {
				return err
			}
			if len(params)>1 {
				m := api.GetMethod(params[1])
				if m != nil {
					bs, err := JSONMarshalIndent(m)
					if err != nil {
						return err
					}
					fmt.Println(string(bs))
				} else {
					return errors.NotFoundError.Errorf("MethodNotFound(name=%s)", params[1])
				}
			} else {
				for itr := api.MethodIterator(); itr.Has() ; itr.Next() {
					fmt.Println(itr.Get().String())
				}
			}
			return nil
		default:
			return errors.IllegalArgumentError.Errorf(
				"InvalidToken(token=%s)", token)
		}
		return nil
	}
}

type shortcut struct {
	Prefix []string
	Suffix []string
}

func applyShortcut(scs map[string]shortcut, params []string) []string {
	if sc, ok := scs[params[0]]; ok {
		p := make([]string, 0, len(params)+len(sc.Prefix)+len(sc.Suffix)-1)
		p = append(p, sc.Prefix...)
		p = append(p, params[1:]...)
		p = append(p, sc.Suffix...)
		return p
	}
	return params
}

var extensionStateShortcuts = map[string]shortcut{
	"prep_base":   {[]string{"dict", "prep_base"}, []string{"obj"}},   // <address>
	"prep_status": {[]string{"dict", "prep_status"}, []string{"obj"}}, // <address>
	"active_prep": {[]string{"array", "active_prep"}, []string{"addr"}},
	"account":     {[]string{"dict", "account_db"}, []string{"obj"}}, // <address>
	"value":       {[]string{"var"}, []string{}},                   // <name>
}

var extensionStageShortcuts = map[string]shortcut{
	"iscore":    {[]string{"rlp", "0x10"}, []string{"obj"}}, // <address>
	"event":     {[]string{"rlp", "0x20"}, []string{"obj"}}, // <offset>/<index>
	"block":     {[]string{"rlp", "0x30"}, []string{"obj"}}, // <offset>
	"validator": {[]string{"rlp", "0x40"}, []string{"obj"}}, // <index>
	"events":    {[]string{"ph", "0x70", "events"}, []string{"obj"}},
	"global":    {[]string{"ph", "0x70", "global"}, []string{"obj"}},
}

var extensionRewardShortcuts = map[string]shortcut{
	"voted":      {[]string{"rlp", "0x10"}, []string{"obj"}}, // <address>
	"delegating": {[]string{"rlp", "0x20"}, []string{"obj"}}, // <address>
	"bonding":    {[]string{"rlp", "0x30"}, []string{"obj"}}, // <address>
	"iscore":     {[]string{"rlp", "0x40"}, []string{"obj"}}, // <address>
}

func showExtensionState(dbase db.Database, hash []byte, params []string) error {
	if len(params) == 0 {
		fmt.Printf("%#x\n", hash)
		return nil
	}
	dbase = icobject.AttachObjectFactory(dbase, icstate.NewObjectImpl)
	snapshot := trie_manager.NewImmutableForObject(dbase, hash, icobject.ObjectType)
	oss := icobject.NewObjectStoreSnapshot(snapshot)
	params = applyShortcut(extensionStateShortcuts, params)
	return showContainerData(oss, params)
}

func showExtensionStage(dbase db.Database, hash []byte, params []string) error {
	if len(params) == 0 {
		fmt.Printf("%#x\n", hash)
		return nil
	}
	dbase = icobject.AttachObjectFactory(dbase, icstage.NewObjectImpl)
	snapshot := trie_manager.NewImmutableForObject(dbase, hash, icobject.ObjectType)
	oss := icobject.NewObjectStoreSnapshot(snapshot)
	params = applyShortcut(extensionStageShortcuts, params)
	return showContainerData(oss, params)
}

func showExtensionReward(dbase db.Database, hash []byte, params []string) error {
	if len(params) == 0 {
		fmt.Printf("%#x\n", hash)
		return nil
	}
	dbase = icobject.AttachObjectFactory(dbase, icreward.NewObjectImpl)
	snapshot := trie_manager.NewImmutableForObject(dbase, hash, icobject.ObjectType)
	oss := icobject.NewObjectStoreSnapshot(snapshot)
	params = applyShortcut(extensionRewardShortcuts, params)
	return showContainerData(oss, params)
}

func showExtension(dbase db.Database, ess state.ExtensionSnapshot, params []string) error {
	if ess == nil {
		return errors.IllegalArgumentError.New("NoExtensionData")
	}
	var hashes [][]byte
	if _, err := codec.BC.UnmarshalFromBytes(ess.Bytes(), &hashes); err != nil {
		return err
	}
	if len(params) < 1 {
		fmt.Printf("State  : <%#x>\n", hashes[0])
		fmt.Printf("Front  : <%#x>\n", hashes[1])
		fmt.Printf("Back   : <%#x>\n", hashes[2])
		fmt.Printf("Reward : <%#x>\n", hashes[3])
		return nil
	}
	param := params[0]
	params = params[1:]
	switch param {
	case "state":
		return showExtensionState(dbase, hashes[0], params)
	case "front":
		return showExtensionStage(dbase, hashes[1], params)
	case "back":
		return showExtensionStage(dbase, hashes[2], params)
	case "reward":
		return showExtensionReward(dbase, hashes[3], params)
	default:
		return errors.IllegalArgumentError.Errorf("UnknownExtensionData(data=%s)", param)
	}
	return nil
}

func showValidators(snapshot state.ValidatorSnapshot, params []string) error {
	if len(params) < 1 {
		fmt.Printf("%#x\n", snapshot.Hash())
		return nil
	}
	param := params[0]
	params = params[1:]
	switch param {
	case "all", "*":
		vl := snapshot.Len()
		for i := 0 ; i<vl ; i++ {
			if validator, ok := snapshot.Get(i); ok {
				fmt.Println(validator.Address())
			}
		}
		return nil
	default:
		if idx, err := strconv.ParseInt(param, 0, 64); err != nil {
			return errors.IllegalArgumentError.Wrapf(err, "InvalidIndex(param=%s)", param)
		} else {
			if validator, ok := snapshot.Get(int(idx)); ok {
				fmt.Println(validator.Address())
				return nil
			} else {
				return errors.IllegalArgumentError.Errorf("OutOfRange(idx=%d,len=%d)", idx, snapshot.Len())
			}
		}
	}
	return nil
}

func showWorld(wss state.WorldSnapshot, params []string) error {
	if len(params) < 1 {
		return errors.IllegalArgumentError.New("" +
			"Address need to be specified")
	}
	param := params[0]
	params = params[1:]
	switch param {
	case "ext":
		return showExtension(wss.Database(), wss.GetExtensionSnapshot(), params)
	case "validators", "val":
		return showValidators(wss.GetValidatorSnapshot(), params)
	default:
		addr, err := common.NewAddressFromString(param)
		if err != nil {
			return errors.IllegalArgumentError.Wrapf(
				err, "InvalidAddress(addr=%s)", param)
		}
		ass := wss.GetAccountSnapshot(addr.ID())
		return showAccount(addr, ass, params)
	}
}

type ExtensionValues struct {
	State  common.HexBytes `json:"state"`
	Front  common.HexBytes `json:"front"`
	Back   common.HexBytes `json:"back"`
	Reward common.HexBytes `json:"reward"`
}

func (ev *ExtensionValues) RLPDecodeSelf(d codec.Decoder) error {
	var bs []byte
	if err := d.Decode(&bs); err != nil {
		return err
	} else {
		var inner [4]common.HexBytes
		if _, err := codec.BC.UnmarshalFromBytes(bs, &inner); err != nil {
			return err
		}
		ev.State = inner[0]
		ev.Front = inner[1]
		ev.Back = inner[2]
		ev.Reward = inner[3]
		return nil
	}
}

type ResultValues struct {
	State          common.HexBytes  `json:"stateHash"`
	PatchReceipts  common.HexBytes  `json:"patchReceipts"`
	NormalReceipts common.HexBytes  `json:"normalReceipts"`
	ExtensionData  *ExtensionValues `json:"extensionData,omitempty"`
}


func getDiffHandlerFor(logger log.Logger, name string) func(op int, key []byte, exp, real trie.Object) {
	return func(op int, key []byte, exp, real trie.Object) {
		switch op {
		case -1:
			logger.Errorf("%s [-] key=%#x value=%+v\n", name, key, exp)
		case 0:
			logger.Errorf("%s [=] key=%#x exp=%+v real=%+v\n", name, key, exp, real)
		case 1:
			logger.Errorf("%s [+] key=%#x value=%+v\n", name, key, real)
		}
	}
}

func showExtensionDiff(dbase db.Database, logger log.Logger, e, r *ExtensionValues) {
	if !bytes.Equal(e.State.Bytes(), r.State.Bytes()) {
		tdb := icobject.AttachObjectFactory(dbase, icstate.NewObjectImpl)
		et := trie_manager.NewImmutableForObject(tdb, e.State.Bytes(), icobject.ObjectType)
		rt := trie_manager.NewImmutableForObject(tdb, r.State.Bytes(), icobject.ObjectType)
		trie_manager.CompareImmutableForObject(et, rt, getDiffHandlerFor(logger, "ext.state"))
	}
	if !bytes.Equal(e.Front.Bytes(), r.Front.Bytes()) {
		tdb := icobject.AttachObjectFactory(dbase, icstage.NewObjectImpl)
		et := trie_manager.NewImmutableForObject(tdb, e.Front.Bytes(), icobject.ObjectType)
		rt := trie_manager.NewImmutableForObject(tdb, r.Front.Bytes(), icobject.ObjectType)
		trie_manager.CompareImmutableForObject(et, rt, getDiffHandlerFor(logger, "ext.front"))
	}
	if !bytes.Equal(e.Back.Bytes(), r.Back.Bytes()) {
		tdb := icobject.AttachObjectFactory(dbase, icstage.NewObjectImpl)
		et := trie_manager.NewImmutableForObject(tdb, e.Front.Bytes(), icobject.ObjectType)
		rt := trie_manager.NewImmutableForObject(tdb, r.Front.Bytes(), icobject.ObjectType)
		trie_manager.CompareImmutableForObject(et, rt, getDiffHandlerFor(logger, "ext.back"))
	}
	if !bytes.Equal(e.Reward.Bytes(), r.Reward.Bytes()) {
		tdb := icobject.AttachObjectFactory(dbase, icreward.NewObjectImpl)
		et := trie_manager.NewImmutableForObject(tdb, e.Reward.Bytes(), icobject.ObjectType)
		rt := trie_manager.NewImmutableForObject(tdb, r.Reward.Bytes(), icobject.ObjectType)
		trie_manager.CompareImmutableForObject(et, rt, getDiffHandlerFor(logger, "ext.reward"))
	}
}

func showResultDiff(dbase db.Database, logger log.Logger, e, r *ResultValues) {
	if !bytes.Equal(e.State.Bytes(), r.State.Bytes()) {
		et := trie_manager.NewImmutableForObject(dbase, e.State.Bytes(), state.AccountType)
		rt := trie_manager.NewImmutableForObject(dbase, r.State.Bytes(), state.AccountType)
		trie_manager.CompareImmutableForObject(et, rt, getDiffHandlerFor(logger, "world"))
	}

	if !bytes.Equal(e.NormalReceipts.Bytes(), r.NormalReceipts.Bytes()) {
		el := txresult.NewReceiptListFromHash(dbase, e.NormalReceipts.Bytes())
		rl := txresult.NewReceiptListFromHash(dbase, r.NormalReceipts.Bytes())
		idx := 0
		for expect, result := el.Iterator(), rl.Iterator(); expect.Has() && result.Has(); _, _, idx = expect.Next(), result.Next(), idx+1 {
			rct1, _ := expect.Get()
			rct2, _ := result.Get()
			if err := rct1.Check(rct2); err != nil {
				rct1js, _ := JSONMarshalIndent(rct1)
				rct2js, _ := JSONMarshalIndent(rct2)
				logger.Errorf("Expected Receipt[%d]:%s", idx, rct1js)
				logger.Errorf("Returned Receipt[%d]:%s", idx, rct2js)
			}
		}
	}
	showExtensionDiff(dbase, logger, e.ExtensionData, r.ExtensionData)
}

func ParseResult(result []byte) (*ResultValues, error) {
	var rv *ResultValues
	if _, err := codec.BC.UnmarshalFromBytes(result, &rv); err != nil {
		return nil, err
	} else {
		return rv, nil
	}
}

func showBlockDetail(blk *Block) error {
	fmt.Printf("Block[%d] - %#x\n", blk.Height(), blk.ID())
	result := blk.Result()
	if len(result) > 0 {
		rv, err := ParseResult(result)
		if err != nil {
			return err
		}
		js, err := JSONMarshalIndent(rv)
		if err != nil {
			return err
		}
		fmt.Printf("- Result : %s\n", js)
	}
	if vh := blk.validators.Hash() ; len(vh) > 0 {
		fmt.Printf("- Validator : %#x\n", vh)
	}
	fmt.Printf("- Block Transactions : %d\n", blk.TxCount())
	fmt.Printf("- Total Transactions : %d\n", blk.TxTotal())
	return nil
}

