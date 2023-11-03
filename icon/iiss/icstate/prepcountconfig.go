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
	"github.com/icon-project/goloop/icon/icmodule"
)

type PRepCountType int

const (
	PRepCountMain  PRepCountType = iota
	PRepCountSub
	PRepCountExtra
)

func (pct PRepCountType) String() string {
	switch pct {
	case PRepCountMain:
		return "main"
	case PRepCountSub:
		return "sub"
	case PRepCountExtra:
		return "extra"
	default:
		return ""
	}
}

func StringToPRepCountType(name string) (PRepCountType, bool) {
	switch name {
	case "main":
		return PRepCountMain, true
	case "sub":
		return PRepCountSub, true
	case "extra":
		return PRepCountExtra, true
	default:
		return -1, false
	}
}

type PRepCountConfig interface {
	MainPReps() int
	ExtraMainPReps() int
	SubPReps() int
	ElectedPReps() int
}

type prepCountConfig struct {
	mainPReps      int
	subPReps       int
	extraMainPReps int
}

func (p prepCountConfig) MainPReps() int {
	return p.mainPReps
}

func (p prepCountConfig) ExtraMainPReps() int {
	return p.extraMainPReps
}

func (p prepCountConfig) SubPReps() int {
	return p.subPReps
}

func (p prepCountConfig) ElectedPReps() int {
	return p.mainPReps + p.subPReps
}

func NewPRepCountConfig(mainPReps, subPReps, extraMainPReps int) PRepCountConfig {
	return prepCountConfig{
		mainPReps:      mainPReps,
		subPReps:       subPReps,
		extraMainPReps: extraMainPReps,
	}
}

func ValidatePRepCountConfig(main, sub, extra int64) error {
	if main <= 0 || main > 1000 {
		return icmodule.IllegalArgumentError.Errorf("InvalidMainPRepCount(%d)", main)
	}
	if sub < 0 || sub > 1000 {
		return icmodule.IllegalArgumentError.Errorf("InvalidSubPRepCount(%d)", sub)
	}
	if extra < 0 || extra > sub || extra > (main-1)/2 {
		return icmodule.IllegalArgumentError.Errorf("InvalidExtraMainPRepCount(%d)", extra)
	}
	return nil
}
