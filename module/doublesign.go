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

package module

const (
	DSTProposal = "proposal"
	DSTVote = "vote"
)

type DoubleSignData interface {
	Type() string
	Height() int64
	Signer() []byte
	Bytes() []byte
	IsConflictWith(other DoubleSignData) bool
}

type DoubleSignContext interface {
	AddressOf(signer []byte) Address
	Hash() []byte
	Bytes() []byte
}

type DoubleSignDataDecoder func (t string, d []byte) (DoubleSignData, error)