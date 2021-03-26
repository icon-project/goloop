/*
 * Copyright 2021 ICON Foundation
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

package lcimporter

import (
	"fmt"

	"github.com/icon-project/goloop/common/log"
)

const (
	CursorUp  = "\x1b[1A"
	ClearLine = "\x1b[2K"
)

var statusDisplay bool

func Statusf(l log.Logger, format string, args ...interface{}) {
	l.Infof(format, args...)
	if l.GetConsoleLevel() < log.InfoLevel {
		if statusDisplay {
			fmt.Print(CursorUp + ClearLine)
		}
		fmt.Printf(format, args...)
		fmt.Print("\n")
		statusDisplay = true
	}
}
