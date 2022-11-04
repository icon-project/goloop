/*
 * Copyright 2020 ICON Foundation
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

package wallet

import (
	"plugin"

	"github.com/icon-project/goloop/common"
	"github.com/icon-project/goloop/common/crypto"
	"github.com/icon-project/goloop/common/errors"
	"github.com/icon-project/goloop/common/log"
	"github.com/icon-project/goloop/module"
)

type walletImpl interface {
	Sign(data []byte) ([]byte, error)
	PublicKey() []byte
}

const builderName = "NewWallet"

type pluginWallet struct {
	walletImpl

	plugin *plugin.Plugin
	addr   module.Address
}

func (w pluginWallet) Address() module.Address {
	return w.addr
}

func OpenPlugin(p string, opts map[string]string) (wallet module.Wallet, ret error) {
	mod, err := plugin.Open(p)
	if err != nil {
		log.Debugf("Fail to open plugin=%s err=%+v", p, err)
		return nil, err
	}
	bdi, err := mod.Lookup(builderName)
	if err != nil {
		log.Debugf("Fail to find %s with plugin=%s err=%+v", builderName, p, err)
		return nil, err
	}
	bd, ok := bdi.(func(params map[string]string) (interface{}, error))
	if !ok {
		return nil, errors.IllegalArgumentError.Errorf(
			"IncompatibleWalletImpl(plugin=%s)", p)
	}
	defer func() {
		rec := recover()
		if rec != nil {
			log.Errorf("Fail to build plugin err=%+v", rec)
			wallet = nil
			ret = errors.ErrIllegalArgument
			return
		}
	}()
	wi, err := bd(opts)
	if err != nil {
		return nil, err
	}
	impl, ok := wi.(walletImpl)
	if !ok {
		return nil, errors.IllegalArgumentError.Errorf(
			"InvalidWalletType(type=%s)", wi)
	}

	pk, err := crypto.ParsePublicKey(impl.PublicKey())
	if err != nil {
		return nil, err
	}

	return &pluginWallet{
		walletImpl: impl,
		plugin:     mod,
		addr:       common.NewAccountAddressFromPublicKey(pk),
	}, nil
}
