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

package ntm

type iconProofPart = secp256k1ProofPart
type iconProof = secp256k1Proof
type iconProofContext = secp256k1ProofContext

func newIconProofContext(keys [][]byte) (*secp256k1ProofContext, error) {
	return newSecp256k1ProofContext(&iconModuleInstance, keys)
}

func newIconProofContextFromBytes(bytes []byte) (*secp256k1ProofContext, error) {
	return newSecp256k1ProofContextFromBytes(&iconModuleInstance, bytes)
}
