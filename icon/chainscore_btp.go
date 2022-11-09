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
	"github.com/icon-project/goloop/common"
	"github.com/icon-project/goloop/common/crypto"
	"github.com/icon-project/goloop/common/intconv"
	"github.com/icon-project/goloop/icon/icmodule"
	"github.com/icon-project/goloop/icon/iiss"
	"github.com/icon-project/goloop/icon/iiss/icstate"
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

const iconDSA = "ecdsa/secp256k1"

func (s *chainScore) Ex_getPRepNodePublicKey(address module.Address) ([]byte, error) {
	if err := s.tryChargeCall(false); err != nil {
		return nil, err
	}
	es, err := s.getExtensionState()
	if err != nil {
		return nil, err
	}
	prep := es.GetPRep(address)
	if prep == nil {
		return nil, icmodule.IllegalArgumentError.New("address is not P-Rep")
	}

	return s.newBTPContext().GetPublicKey(prep.NodeAddress(), iconDSA), nil
}

func (s *chainScore) Ex_setPRepNodePublicKey(pubKey []byte) error {
	return s.setPRepNodePublicKey(nil, pubKey)
}

func (s *chainScore) Ex_registerPRepNodePublicKey(address module.Address, pubKey []byte) error {
	return s.setPRepNodePublicKey(address, pubKey)
}

func (s *chainScore) setPRepNodePublicKey(address module.Address, pubKey []byte) error {
	if s.from.IsContract() {
		return scoreresult.New(module.StatusAccessDenied, "NoPermission")
	}
	if len(pubKey) == 0 {
		return icmodule.IllegalArgumentError.New("Invalid pubKey")
	}
	es, err := s.getExtensionState()
	if err != nil {
		return err
	}
	register := true
	if address == nil {
		register = false
		address = s.from
	}
	prep := es.GetPRep(address)
	if prep == nil {
		return icmodule.IllegalArgumentError.New("address is not P-Rep")
	}
	nodeAddress := prep.NodeAddress()

	pk, err := crypto.ParsePublicKey(pubKey)
	if err != nil {
		return icmodule.IllegalArgumentError.Wrap(err, "Failed to parse public key")
	}
	pubKeyAddr := common.NewAccountAddressFromPublicKey(pk)

	bc := s.newBTPContext()
	bs, err := s.getBTPState()
	if err != nil {
		return err
	}

	if register {
		if v := bc.GetPublicKey(pubKeyAddr, iconDSA); v != nil {
			return icmodule.IllegalArgumentError.New("There is public key already. To update public key, use setPRepNodePublicKey")
		}
		if !pubKeyAddr.Equal(nodeAddress) {
			return icmodule.IllegalArgumentError.Errorf(
				"Public key and node address of P-Rep do not match. %s!=%s", pubKeyAddr, nodeAddress)
		}
	} else {
		if !pubKeyAddr.Equal(nodeAddress) {
			// remove old public key
			if err = bs.SetPublicKey(bc, nodeAddress, iconDSA, []byte{}); err != nil {
				return err
			}
			// update node address of P-Rep
			prepInfo := &icstate.PRepInfo{Node: pubKeyAddr}
			if err = es.SetPRep(s.newCallContext(s.cc), prepInfo, true); err != nil {
				return err
			}
		}
	}

	if err = bs.SetPublicKey(bc, pubKeyAddr, iconDSA, pubKey); err != nil {
		return err
	}
	prep.SetDSAMask(bc.GetPublicKeyMask(pubKeyAddr))
	if err = es.OnSetPublicKey(s.newCallContext(s.cc), prep.Owner(), bc.GetDSAIndex(iconDSA)); err != nil {
		return err
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
			var es *iiss.ExtensionStateImpl
			if es, err = s.getExtensionState(); err != nil {
				return 0, err
			}
			if err = es.OnOpenBTPNetwork(s.newCallContext(s.cc), bc, networkTypeName); err != nil {
				return 0, err
			}
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
		return icmodule.IllegalArgumentError.New("Invalid BTP message")
	}
	if bs, err := s.getBTPState(); err != nil {
		return err
	} else {
		nid := networkId.Int64()
		sn, err := bs.HandleMessage(s.newBTPContext(), s.from, nid)
		if err != nil {
			return err
		}
		s.cc.OnBTPMessage(nid, message)
		s.cc.OnEvent(state.SystemAddress,
			[][]byte{
				[]byte("BTPMessage(int,int)"),
				intconv.Int64ToBytes(nid),
				intconv.Int64ToBytes(sn),
			},
			nil,
		)
		return nil
	}
}
