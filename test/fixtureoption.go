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
	"github.com/icon-project/goloop/common/db"
	"github.com/icon-project/goloop/module"
)

type FixtureOption func(cf *FixtureConfig) *FixtureConfig

func UseConfig(cf2 *FixtureConfig) FixtureOption {
	return func(cf *FixtureConfig) *FixtureConfig {
		return cf.Override(cf2)
	}
}

func UseDB(dbase db.Database) FixtureOption {
	return UseConfig(&FixtureConfig{
		Dbase: func() db.Database{
			return dbase
		},
	})
}

// AddValidatorNodes option makes n validators and the first validator becomes
// default node
func AddValidatorNodes(n int) FixtureOption {
	return UseConfig(&FixtureConfig{AddValidatorNodes: n})
}

func UseGenesis(gs string) FixtureOption {
	return UseConfig(&FixtureConfig{Genesis: gs})
}

func UseGenesisStorage(gs module.GenesisStorage) FixtureOption {
	return UseConfig(&FixtureConfig{
		Genesis:        string(gs.Genesis()),
		GenesisStorage: gs,
	})
}

func UseWallet(w module.Wallet) FixtureOption {
	return UseConfig(&FixtureConfig{Wallet: w})
}

func AddDefaultNode(v bool) FixtureOption {
	return UseConfig(&FixtureConfig{AddDefaultNode: &v})
}
