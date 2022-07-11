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

package test

import (
	"fmt"
	"testing"

	"github.com/icon-project/goloop/common/wallet"
	"github.com/icon-project/goloop/module"
)

type Fixture struct {
	// default node
	*Node
	BaseConfig *FixtureConfig

	// all nodes
	Nodes      []*Node
}

func NewFixture(t *testing.T, o ...FixtureOption) *Fixture {
	cf := NewFixtureConfig(t, o...)
	f := &Fixture{
		BaseConfig: cf,
	}
	var gs string
	if cf.AddValidatorNodes > 0 {
		wallets := make([]module.Wallet, cf.AddValidatorNodes)
		for i := range wallets {
			wallets[i] = wallet.New()
		}
		var validators string
		for i, w := range wallets {
			if i > 0 {
				validators += ", "
			}
			validators += fmt.Sprintf(`"%s"`, w.Address())
		}
		gs = fmt.Sprintf(`{
			"accounts": [
				{
					"name" : "treasury",
					"address" : "hx1000000000000000000000000000000000000000",
					"balance" : "0x0"
				},
				{
					"name" : "god",
					"address" : "hx0000000000000000000000000000000000000000",
					"balance" : "0x0"
				}
			],
			"message": "",
			"nid" : "0x1",
			"chain" : {
				"validatorList" : [ %s ]
			}
		}`, validators)
		for i := range wallets {
			f.AddNode(UseGenesis(gs), UseWallet(wallets[i]))
		}
	}
	if *cf.AddDefaultNode {
		node := f.AddNode(UseGenesis(gs))
		f.Node = node
	}
	return f
}

func (f *Fixture) AddNode(o ...FixtureOption) *Node {
	eo := make([]FixtureOption, 0, len(o)+1)
	eo = append(eo, UseConfig(f.BaseConfig))
	eo = append(eo, o...)
	node := NewNode(f.BaseConfig.T, eo...)
	f.Nodes = append(f.Nodes, node)
	if f.Node == nil {
		f.Node = node
	}
	return node
}

func (f *Fixture) AddNodes(n int, o ...FixtureOption) []*Node {
	nodes := make([]*Node, n)
	for i:=0; i<n; i++ {
		nodes[i] = f.AddNode(o...)
	}
	return nodes
}

func (f *Fixture) Close() {
	for _, n := range f.Nodes {
		n.Close()
	}
}
