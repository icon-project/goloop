/*
 * Copyright 2022 ICON Foundation
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

package btp

import "github.com/icon-project/goloop/module"

type networkType struct {
	uid                  string
	nextProofContextHash []byte
	nextProofContext     []byte
	openNetworkIDs       []int64
}

func (nt *networkType) UID() string {
	return nt.uid
}

func (nt *networkType) NextProofContextHash() []byte {
	return nt.nextProofContextHash
}

func (nt *networkType) NextProofContext() []byte {
	return nt.nextProofContext
}

func (nt *networkType) OpenNetworkIDs() []int64 {
	return nt.openNetworkIDs
}

type network struct {
	name                    string
	owner                   module.Address
	networkTypeID           int64
	open                    bool
	nextMessageSN           int64
	nextProofContextChanged bool
	prevNetworkSectionHash  []byte
	lastNetworkSectionHash  []byte
}

func (nw *network) Name() string {
	return nw.name
}

func (nw *network) Owner() module.Address {
	return nw.owner
}

func (nw *network) NetworkTypeID() int64 {
	return nw.networkTypeID
}

func (nw *network) Open() bool {
	return nw.open
}

func (nw *network) NextMessageSN() int64 {
	return nw.nextMessageSN
}

func (nw *network) NextProofContextChanged() bool {
	return nw.nextProofContextChanged
}

func (nw *network) PrevNetworkSectionHash() []byte {
	return nw.prevNetworkSectionHash
}

func (nw *network) LastNetworkSectionHash() []byte {
	return nw.lastNetworkSectionHash
}
