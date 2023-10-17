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
 */

package icstate

import (
	"github.com/icon-project/goloop/common/log"
)

type MainPRepType int
const (
	MainPRepNormal MainPRepType = iota
	MainPRepExtra
	MainPRepAll
)

type PRepCountConfig interface {
	MainPReps(mpt MainPRepType) int
	SubPReps() int
	ElectedPReps() int
}

type prepCountConfig struct {
	mainPReps int
	subPReps int
	extraMainPReps int
}

func (p prepCountConfig) MainPReps(mpt MainPRepType) int {
	switch mpt {
	case MainPRepNormal:
		return p.mainPReps
	case MainPRepExtra:
		return p.extraMainPReps
	case MainPRepAll:
		return p.mainPReps + p.extraMainPReps
	default:
		log.Panicf("UnknownMainPRepType(%d)", mpt)
	}
	return -1
}

func (p prepCountConfig) SubPReps() int {
	return p.subPReps
}

func (p prepCountConfig) ElectedPReps() int {
	return p.mainPReps + p.subPReps
}

func NewPRepCountConfig(mainPReps, subPReps, extraMainPReps int) PRepCountConfig {
	return prepCountConfig{
		mainPReps: mainPReps,
		subPReps: subPReps,
		extraMainPReps: extraMainPReps,
	}
}

func IsPRepCountConfigValid(cfg PRepCountConfig) bool {
	main := cfg.MainPReps(MainPRepNormal)
	sub := cfg.SubPReps()
	extra := cfg.MainPReps(MainPRepExtra)

	return (main > 0 && main <= 1000) &&
		(sub >= 0 && sub <= 1000) &&
		(extra >= 0 && extra <= sub)
}
