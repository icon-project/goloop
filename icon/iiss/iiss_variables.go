package iiss

import "math/big"

const iissDayBlock = 30 * 60 * 24

var RPoint = big.NewFloat(0.7)
var LockMin = big.NewInt(iissDayBlock * 5)
var LockMax = big.NewInt(iissDayBlock * 20)
