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

package fastsync

import (
	"bytes"

	"github.com/icon-project/goloop/common/errors"
	"github.com/icon-project/goloop/module"
)

type blockByHeightAndHashRequest struct {
	fsm      Manager
	h        int64
	hash     []byte
	brCh     chan BlockResult
	errCh    chan error
	cancelCh <-chan struct{}
}

func (r *blockByHeightAndHashRequest) OnBlock(br BlockResult) {
	if bytes.Equal(br.Block().Hash(), r.hash) {
		br.Consume()
		r.brCh <- br
	} else {
		br.Reject()
		r.errCh <- errors.Errorf("unexpected hash height=%d expHash=%x actHash=%x", r.h, r.hash, br.Block().Hash())
	}
}

func (r *blockByHeightAndHashRequest) OnEnd(err error) {
	if err != nil {
		r.errCh <- err
	}
}

func FetchBlockByHeightAndHash(
	fsm Manager, h int64, hash []byte, cancelCh <-chan struct{},
) (module.BlockData, []byte, error) {
	r := &blockByHeightAndHashRequest{
		fsm:      fsm,
		h:        h,
		hash:     hash,
		brCh:     make(chan BlockResult, 1),
		errCh:    make(chan error, 2),
		cancelCh: cancelCh,
	}
	canceler, err := fsm.FetchBlocks(h, h, r)
	if err != nil {
		return nil, nil, err
	}

	select {
	case br := <-r.brCh:
		return br.Block(), br.Votes(), nil
	case err := <-r.errCh:
		return nil, nil, err
	case <-r.cancelCh:
		canceler()
		return nil, nil, errors.Errorf("blockRequestCanceled height=%d hash=%x", r.h, r.hash)
	}
}
