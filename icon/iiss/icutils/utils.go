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

package icutils

import (
	"math/big"

	"github.com/icon-project/goloop/common"
	"github.com/icon-project/goloop/common/errors"
	"github.com/icon-project/goloop/common/intconv"
	"github.com/icon-project/goloop/common/log"
	"github.com/icon-project/goloop/icon/icmodule"
	"github.com/icon-project/goloop/module"
	"github.com/icon-project/goloop/service/contract"
	"github.com/icon-project/goloop/service/scoredb"
	"github.com/icon-project/goloop/service/state"

	"regexp"
	"strconv"
	"strings"
)

const (
	SchemePattern   = `^(http:\/\/|https:\/\/)`
	HostNamePattern = `(localhost|(?:[\w\d](?:[\w\d-]{0,61}[\w\d])\.)+[\w\d][\w\d-]{0,61}[\w\d])`
	IPv4Pattern     = `(([0-9]|[1-9][0-9]|1[0-9]{2}|2[0-4][0-9]|25[0-5])\.){3}([0-9]|[1-9][0-9]|1[0-9]{2}|2[0-4][0-9]|25[0-5])`
	PortPattern     = `(:[0-9]{1,5})?`
	PathPattern     = `(\/\S*)?$`
	EmailPattern    = `^[a-zA-Z0-9]+(\.[a-zA-Z0-9]+)*@` + HostNamePattern + `$`

	PortMax       = 65536
	EmailLocalMax = 64
	EmailMax      = 254
)

var (
	websiteDNTemplate    = regexp.MustCompile(SchemePattern + HostNamePattern + PortPattern + PathPattern)
	websiteIPv4Template  = regexp.MustCompile(SchemePattern + IPv4Pattern + PortPattern + PathPattern)
	emailTemplate        = regexp.MustCompile(EmailPattern)
	endpointDNTemplate   = regexp.MustCompile(IPv4Pattern + PortPattern)
	endpointIPv4Template = regexp.MustCompile(HostNamePattern + PortPattern)
)

var BigIntICX = big.NewInt(1_000_000_000_000_000_000)

func ToLoop(icx int) *big.Int {
	return ToDecimal(icx, 18)
}

// ToDecimal
func ToDecimal(x, y int) *big.Int {
	if y < 0 {
		return nil
	}
	ret := big.NewInt(int64(x))
	return ret.Mul(ret, Pow10(y))
}

func Pow10(n int) *big.Int {
	if n < 0 {
		return nil
	}
	if n == 0 {
		return big.NewInt(1)
	}
	ret := big.NewInt(10)
	return ret.Exp(ret, big.NewInt(int64(n)), nil)
}

func MergeMaps(maps ...map[string]interface{}) map[string]interface{} {
	size := len(maps)
	if size == 0 {
		return nil
	}

	ret := maps[0]
	for i := 1; i < size; i++ {
		for k, v := range maps[i] {
			ret[k] = v
		}
	}

	return ret
}

func ToKey(o interface{}) string {
	switch o := o.(type) {
	case module.Address:
		return string(o.Bytes())
	case []byte:
		return string(o)
	default:
		panic(errors.Errorf("Unsupported type: %v", o))
	}
}

func EqualAddress(a1 module.Address, a2 module.Address) bool {
	if a1 == a2 {
		return true
	}

	if a1 != nil {
		return a1.Equal(a2)
	} else if a2 != nil {
		return a2.Equal(a1)
	}

	return false
}

func GetTotalSupply(ws state.WorldState) *big.Int {
	as := ws.GetAccountState(state.SystemID)
	tsVar := scoredb.NewVarDB(as, state.VarTotalSupply)
	return tsVar.BigInt()
}

func IncreaseTotalSupply(ws state.WorldState, amount *big.Int) (*big.Int, error) {
	as := ws.GetAccountState(state.SystemID)
	tsVar := scoredb.NewVarDB(as, state.VarTotalSupply)
	ts := new(big.Int).Add(tsVar.BigInt(), amount)
	if ts.Sign() < 0 {
		return nil, errors.Errorf("TotalSupply < 0")
	}
	return ts, tsVar.Set(ts)
}

func DecreaseTotalSupply(ws state.WorldState, amount *big.Int) (*big.Int, error) {
	return IncreaseTotalSupply(ws, new(big.Int).Neg(amount))
}

func OnBurn(cc contract.CallContext, address module.Address, amount, ts *big.Int) {
	rev := cc.Revision().Value()
	if rev < icmodule.RevisionBurnV2 {
		var burnSig string
		if rev < icmodule.RevisionFixBurnEventSignature {
			burnSig = "ICXBurned"
		} else {
			burnSig = "ICXBurned(int)"
		}
		cc.OnEvent(state.SystemAddress,
			[][]byte{[]byte(burnSig)},
			[][]byte{intconv.BigIntToBytes(amount)},
		)
	} else {
		cc.OnEvent(state.SystemAddress,
			[][]byte{[]byte("ICXBurnedV2(Address,int,int)"), address.Bytes()},
			[][]byte{intconv.BigIntToBytes(amount), intconv.BigIntToBytes(ts)},
		)
	}
}

func Min(value1, value2 int) int {
	if value1 < value2 {
		return value1
	} else {
		return value2
	}
}

func BigInt2HexInt(value *big.Int) *common.HexInt {
	h := new(common.HexInt)
	h.Set(value)
	return h
}

func ValidateRange(oldValue *big.Int, newValue *big.Int, minPct int, maxPct int) error {
	diff := new(big.Int).Sub(oldValue, newValue)
	switch diff.Sign() {
	case 1:
		threshold := new(big.Int).Mul(oldValue, new(big.Int).SetInt64(int64(100-minPct)))
		threshold.Div(threshold, new(big.Int).SetInt64(100))
		if newValue.CmpAbs(threshold) == -1 {
			return errors.Errorf("Out of range: %s < %s", newValue, threshold)
		}
	case -1:
		threshold := new(big.Int).Mul(oldValue, new(big.Int).SetInt64(int64(100+maxPct)))
		threshold.Div(threshold, new(big.Int).SetInt64(100))
		if newValue.CmpAbs(threshold) == 1 {
			return errors.Errorf("Out of range: %s > %s", newValue, threshold)
		}
	}
	return nil
}

func NewIconLogger(logger log.Logger) log.Logger {
	if logger == nil {
		return nil
	}
	return logger.WithFields(log.Fields{log.FieldKeyModule: "ICON"})
}

func ValidateEndpoint(endpoint string) error {
	if len(endpoint) == 0 {
		return nil
	}

	networkInfo := strings.Split(endpoint, ":")

	if len(networkInfo) != 2 {
		return errors.Errorf("Invalid endpoint format, must have port info.")
	}

	port, err := strconv.Atoi(networkInfo[1])
	if err != nil {
		return err
	}

	// port validate
	if !(0 < port && port < PortMax) {
		return errors.Errorf("Invalid endpoint format, Port out of range.")
	}

	endpointLower := strings.ToLower(endpoint)
	if !(endpointDNTemplate.MatchString(endpointLower) || endpointIPv4Template.MatchString(endpointLower)) {
		return errors.Errorf("Invalid endpoint format")
	}

	return nil
}

func ValidateURL(url string) error {
	if len(url) == 0 {
		return nil
	}

	websiteURI := strings.ToLower(url)
	if !(websiteDNTemplate.MatchString(websiteURI) || websiteIPv4Template.MatchString(websiteURI)) {
		return errors.Errorf("Invalid websiteURL format")
	}

	return nil
}

func ValidateEmail(email string, revision int) error {
	if len(email) == 0 {
		return nil
	}

	if revision < icmodule.RevisionFixEmailValidation {
		if !emailTemplate.MatchString(email) {
			return errors.Errorf("Invalid Email format")
		}
	} else {
		index := strings.LastIndex(email, "@")
		length := len(email)

		beforeCheck := 1 <= index && index <= EmailLocalMax
		afterCheck := index+1 < length && length <= EmailMax

		if !(beforeCheck && afterCheck) {
			return errors.Errorf("Invalid Email format")
		}
	}

	return nil
}
