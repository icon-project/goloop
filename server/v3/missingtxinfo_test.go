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

package v3

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetMissingTxLocator(t *testing.T) {
	for _, txHash := range subTxHashes {
		height, txIndex, ok := iconMissedTransactions.GetLocationOf([]byte(txHash))
		assert.True(t, height > int64(0))
		assert.True(t, txIndex >= 0)
		assert.True(t, ok)
	}

	dummyTxHash := make([]byte, 32)
	height, txIndex, ok := iconMissedTransactions.GetLocationOf(dummyTxHash)
	assert.Equal(t, int64(-1), height)
	assert.Equal(t, -1, txIndex)
	assert.False(t, ok)
}

func TestReplaceMissingTxHash(t *testing.T) {
	heights := []int64{43244423, 43271612, 43271612, 43292491, 43292491}

	for i, txHash := range missingTxHashes {
		subTxHash := string(iconMissedTransactions.ReplaceID(heights[i], []byte(txHash)))
		assert.Equal(t, subTxHashes[i], subTxHash)
		assert.Equal(t, subTxHash[31]+1, txHash[31])
		assert.Equal(t, subTxHash[:31], txHash[:31])
	}

	// NoMissingTxHashes
	height := int64(1234)
	for _, noMissingTxHash := range subTxHashes {
		txHash := string(iconMissedTransactions.ReplaceID(height, []byte(noMissingTxHash)))
		assert.Equal(t, txHash, noMissingTxHash)
	}

	// MissingTxHash but height is different
	for _, missingTxHash := range missingTxHashes {
		txHash := string(iconMissedTransactions.ReplaceID(height, []byte(missingTxHash)))
		assert.Equal(t, txHash, missingTxHash)
	}
}
