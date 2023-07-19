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
	"reflect"
	"regexp"
	"strconv"
	"strings"

	"github.com/biter777/countries"

	"github.com/icon-project/goloop/common"
	"github.com/icon-project/goloop/common/errors"
	"github.com/icon-project/goloop/common/log"
	"github.com/icon-project/goloop/icon/icmodule"
	"github.com/icon-project/goloop/module"
)

const (
	SchemePattern   = `(http:\/\/|https:\/\/)`
	HostNamePattern = `(localhost|(?:[\w\d](?:[\w\d-]{0,61}[\w\d])\.)+[\w\d][\w\d-]{0,61}[\w\d])`
	IPv4Pattern     = `(([0-9]|[1-9][0-9]|1[0-9]{2}|2[0-4][0-9]|25[0-5])\.){3}([0-9]|[1-9][0-9]|1[0-9]{2}|2[0-4][0-9]|25[0-5])`
	PortPattern     = `(:[0-9]{1,5})?`
	PathPattern     = `(\/\S*)?`
	EmailPattern    = `[a-zA-Z0-9]+(\.[a-zA-Z0-9]+)*@` + HostNamePattern

	PortMax       = 65536
	EmailLocalMax = 64
	EmailMax      = 254
)

var (
	websiteDNTemplate    = regexp.MustCompile(`^` + SchemePattern + HostNamePattern + PortPattern + PathPattern + `$`)
	websiteIPv4Template  = regexp.MustCompile(`^` + SchemePattern + IPv4Pattern + PortPattern + PathPattern + `$`)
	emailTemplate        = regexp.MustCompile(`^` + EmailPattern + `$`)
	endpointDNTemplate   = regexp.MustCompile(`^` + IPv4Pattern + PortPattern + `$`)
	endpointIPv4Template = regexp.MustCompile(`^` + HostNamePattern + PortPattern + `$`)
)

var BigIntICX = big.NewInt(1_000_000_000_000_000_000)

func ToLoop(icx int) *big.Int {
	return ToDecimal(icx, 18)
}

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

	ret := make(map[string]interface{})
	for i := 0; i < size; i++ {
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

func Min(value1, value2 int) int {
	if value1 < value2 {
		return value1
	} else {
		return value2
	}
}

func MinBigInt(v0, v1 *big.Int) *big.Int {
	if v0.Cmp(v1) < 0 {
		return v0
	}
	return v1
}

func BigInt2HexInt(value *big.Int) *common.HexInt {
	h := new(common.HexInt)
	h.Set(value)
	return h
}

func ValidateRange(oldValue *big.Int, newValue *big.Int, minPct int, maxPct int) error {
	switch oldValue.Cmp(newValue) {
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
	fields := log.Fields{log.FieldKeyModule: "ICON"}
	if logger == nil {
		return log.WithFields(fields)
	}
	return logger.WithFields(fields)
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

func ValidateCountryAlpha3(alpha3 string) error {
	code := countries.ByName(alpha3)
	if code == countries.Unknown {
		return errors.IllegalArgumentError.Errorf("UnknownCountry(alpha3=%s)", alpha3)
	}
	if code.Alpha3() != strings.ToUpper(alpha3) {
		return errors.IllegalArgumentError.Errorf("UseAlpha3(alpha3=%s,name=%s)",
			code.Alpha3(), alpha3)
	}
	return nil
}

func ICXToIScore(icx *big.Int) *big.Int {
	return new(big.Int).Mul(icx, icmodule.BigIntIScoreICXRatio)
}

func IScoreToICX(iScore *big.Int) *big.Int {
	return new(big.Int).Div(iScore, icmodule.BigIntIScoreICXRatio)
}

func IsNil(i interface{}) bool {
	return i == nil || reflect.ValueOf(i).IsNil()
}

// CalcPower calculates the amount of power with bondRequirement, bonded and voted (= bonded + delegated)
func CalcPower(br icmodule.Rate, bonded, voted *big.Int) *big.Int {
	if br == 0 {
		// when bondRequirement is 0, it means no threshold for BondedRequirement,
		// so it returns 100% of totalVoted.
		// And it should not be divided by 0 in the following code that could occurs Panic.
		return voted
	}
	power := new(big.Int).Mul(bonded, br.DenomBigInt())
	power.Div(power, br.NumBigInt())
	return MinBigInt(power, voted)
}