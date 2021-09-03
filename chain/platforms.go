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

package chain

import (
	"github.com/icon-project/goloop/chain/base"
	"github.com/icon-project/goloop/common/errors"
	"github.com/icon-project/goloop/service/platform/basic"
)

type PlatformFactory func(base string, cid int) (base.Platform, error)

var platformFactories = map[string]PlatformFactory{
	"basic": func(base string, cid int) (base.Platform, error) {
		return basic.Platform, nil
	},
}

func RegisterPlatform(name string, factory PlatformFactory) {
	platformFactories[name] = factory
}

func NewPlatform(name string, base string, cid int) (base.Platform, error) {
	if len(name) == 0 {
		name = "basic"
	}
	if factory, ok := platformFactories[name]; ok {
		return factory(base, cid)
	}
	return nil, errors.NotFoundError.Errorf("PlatformNotFound(name=%s)", name)
}
