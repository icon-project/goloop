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

package lcimporter

import (
	"github.com/icon-project/goloop/chain/base"
	"github.com/icon-project/goloop/common"
	"github.com/icon-project/goloop/icon/blockv0/lcstore"
	"github.com/icon-project/goloop/service/eeproxy"
)

type Config struct {
	Validators  []*common.Address   `json:"validators"`
	StoreURI    string              `json:"store_uri"`
	MaxRPS      int                 `json:"max_rps"`
	CacheConfig lcstore.CacheConfig `json:"cache_config"`
	BaseDir  string
	Platform base.Platform
	ProxyMgr eeproxy.Manager
}
