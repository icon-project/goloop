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

package icon

import (
	"github.com/icon-project/goloop/btp/ntm"
	"github.com/icon-project/goloop/common"
	"github.com/icon-project/goloop/common/intconv"
	"github.com/icon-project/goloop/icon/iiss"
	"github.com/icon-project/goloop/module"
	"github.com/icon-project/goloop/service/scoreresult"
	"github.com/icon-project/goloop/service/state"
)

func (s *chainScore) getBTPState() (*state.BTPStateImpl, error) {
	btpState := s.cc.GetBTPState()
	if btpState == nil {
		return nil, scoreresult.UnknownFailureError.Errorf("BTP state is nil")
	}
	return btpState.(*state.BTPStateImpl), nil
}

func (s *chainScore) newBTPContext() state.BTPContext {
	store := s.cc.GetAccountState(state.SystemID)
	return state.NewBTPContext(s.cc, store)
}

func (s *chainScore) Ex_getBTPNetworkTypeID(name string) (int64, error) {
	if err := s.tryChargeCall(false); err != nil {
		return 0, err
	}
	return s.newBTPContext().GetNetworkTypeIDByName(name), nil
}

const iconBTPUID = "icon"
const iconDSA = "ecdsa/secp256k1"

func (s *chainScore) Ex_getNodePublicKey(address module.Address) ([]byte, error) {
	if err := s.tryChargeCall(false); err != nil {
		return nil, err
	}
	pubKey, _ := s.newBTPContext().GetPublicKey(address, iconDSA, true)
	return pubKey, nil
}

func (s *chainScore) Ex_setNodePublicKey(prep module.Address, pubKey []byte, update bool) error {
	if s.from.IsContract() {
		return scoreresult.New(module.StatusAccessDenied, "NoPermission")
	}
	if len(pubKey) == 0 {
		return scoreresult.New(module.StatusInvalidParameter, "Invalid pubKey")
	}
	var err error
	var es *iiss.ExtensionStateImpl
	if es, err = s.getExtensionState(); err != nil {
		return err
	}
	bc := s.newBTPContext()
	if jso, err := es.GetPRepInJSON(prep, s.cc.BlockHeight()); err != nil {
		return scoreresult.New(module.StatusInvalidParameter, "prep is not P-Rep")
	} else {
		if update {
			if !s.from.Equal(prep) {
				return scoreresult.New(module.StatusAccessDenied, "Only the P-Rep can update its own public key")
			}
		} else {
			if len(pubKey) > 0 {
				mod := ntm.ForUID(iconBTPUID)
				id, err := mod.AddressFromPubKey(pubKey)
				if err != nil {
					return err
				}
				addr := common.MustNewAddress(id)
				if !addr.Equal(jso["nodeAddress"].(module.Address)) {
					return scoreresult.Errorf(module.StatusInvalidParameter,
						"Public key and node address of P-Rep do not match. %s!=%s", addr, jso["nodeAddress"])
				}
				if v, ok := bc.GetPublicKey(addr, iconDSA, true); v != nil && ok {
					return scoreresult.New(module.StatusInvalidParameter,
						"There is public key already. To update public key, set update true")
				}
			}
		}
	}

	if bs, err := s.getBTPState(); err != nil {
		return err
	} else {
		if err = bs.SetPublicKey(bc, s.from, iconDSA, pubKey); err != nil {
			return err
		}
	}
	return nil
}

func (s *chainScore) Ex_openBTPNetwork(networkTypeName string, name string, owner module.Address) (int64, error) {
	if err := s.checkGovernance(true); err != nil {
		return 0, err
	}
	if bs, err := s.getBTPState(); err != nil {
		return 0, err
	} else {
		bc := s.newBTPContext()
		ntActivated := false
		if bc.GetNetworkTypeIDByName(networkTypeName) <= 0 {
			ntActivated = true
		}
		ntid, nid, err := bs.OpenNetwork(bc, networkTypeName, name, owner)
		if err != nil {
			return 0, err
		}
		if ntActivated {
			s.cc.OnEvent(state.SystemAddress,
				[][]byte{
					[]byte("BTPNetworkTypeActivated(str,int)"),
					[]byte(networkTypeName),
					intconv.Int64ToBytes(ntid),
				},
				nil,
			)
		}
		s.cc.OnEvent(state.SystemAddress,
			[][]byte{
				[]byte("BTPNetworkOpened(int,int)"),
				intconv.Int64ToBytes(ntid),
				intconv.Int64ToBytes(nid),
			},
			nil,
		)
		return nid, nil
	}
}

func (s *chainScore) Ex_closeBTPNetwork(id *common.HexInt) error {
	if err := s.checkGovernance(true); err != nil {
		return err
	}
	nid := id.Int64()
	if bs, err := s.getBTPState(); err != nil {
		return err
	} else {
		if ntid, err := bs.CloseNetwork(s.newBTPContext(), nid); err != nil {
			return err
		} else {
			s.cc.OnEvent(state.SystemAddress,
				[][]byte{
					[]byte("BTPNetworkClosed(int,int)"),
					intconv.Int64ToBytes(ntid),
					intconv.Int64ToBytes(nid),
				},
				nil,
			)
		}
	}
	return nil
}

func (s *chainScore) Ex_sendBTPMessage(networkId *common.HexInt, message []byte) error {
	if err := s.tryChargeCall(false); err != nil {
		return err
	}
	if len(message) == 0 {
		return scoreresult.New(module.StatusInvalidParameter, "Invalid BTP message")
	}
	if bs, err := s.getBTPState(); err != nil {
		return err
	} else {
		nid := networkId.Int64()
		if err = bs.HandleMessage(s.newBTPContext(), s.from, nid); err != nil {
			return err
		}
		s.cc.OnBTPMessage(nid, message)
		return nil
	}
}
