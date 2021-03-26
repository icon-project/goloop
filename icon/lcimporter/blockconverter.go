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
	"github.com/icon-project/goloop/common/db"
	"github.com/icon-project/goloop/common/errors"
	"github.com/icon-project/goloop/common/log"
	"github.com/icon-project/goloop/icon"
	"github.com/icon-project/goloop/icon/blockv0"
	"github.com/icon-project/goloop/module"
	"github.com/icon-project/goloop/service"
	"github.com/icon-project/goloop/service/contract"
	"github.com/icon-project/goloop/service/eeproxy"
)

const (
	ContractPath = "contract"
	EESocketPath = "ee.sock"
)
const (
	FlagExecutor = "executor"
)

const (
	KeyLastBlockHeight   = "block.lastHeight"
)

const (
	JSONByHash db.BucketID = "J"
)

type BlockConverter struct {
	baseDir  string
	cs       Store
	database db.Database
	cm       contract.ContractManager
	em       eeproxy.Manager
	chain    module.Chain
	log      log.Logger
	plt      service.Platform
	trace    log.Logger

	sHeight int64

	jsBucket    db.Bucket
	blkIndex    db.Bucket
	blkByID     db.Bucket
	chainBucket db.Bucket
}

type Transition struct {
	module.Transition
	//Block *Block
}

type Store interface {
	GetRepsByHash(id []byte) (*blockv0.RepsList, error)
	GetBlockByHeight(height int) (blockv0.Block, error)
	GetReceipt(id []byte) (module.Receipt, error)
}

type GetTPSer interface {
	GetTPS() float32
}

func NewBlockConverter(chain module.Chain, cs Store, data string) (*BlockConverter, error) {
	database := chain.Database()
	logger := chain.Logger()
	plt, err := icon.NewPlatform(data, chain.CID())
	if err != nil {
		return nil, errors.Wrap(err, "NewPlatformFailure")
	}
	jsBucket, err := database.GetBucket(JSONByHash)
	if err != nil {
		return nil, errors.Wrap(err, "FailureInGetBucketForJSON")
	}
	blkIndex, err := database.GetBucket(db.BlockHeaderHashByHeight)
	if err != nil {
		return nil, errors.Wrap(err, "FailureInBucket(bucket=HashByHeight)")
	}
	blkByID, err := database.GetBucket(db.BlockV1ByHash)
	if err != nil {
		return nil, errors.Wrap(err, "FailureInBucket(bucket=BlockV1ByHash)")
	}
	chainBucket, err := database.GetBucket(db.ChainProperty)
	if err != nil {
		return nil, errors.Wrap(err, "FailureInBucket(bucket=ChainProperty)")
	}
	ex := &BlockConverter{
		baseDir:     data,
		cs:          cs,
		log:         logger,
		chain:       chain,
		plt:         plt,
		jsBucket:    jsBucket,
		blkIndex:    blkIndex,
		blkByID:     blkByID,
		chainBucket: chainBucket,
	}
	ex.trace = logger.WithFields(log.Fields{
		log.FieldKeyModule: "TRACE",
	})
	ex.database = db.WithFlags(database, db.Flags{
		FlagExecutor: ex,
	})
	return ex, nil
}

const ChanBuf = 2048

func (e *BlockConverter) Start(from, to int64) (<- chan interface{}, error) {
	chn := make(chan interface{}, ChanBuf)
	return chn, nil
}

// Rebase re-bases blocks and returns a channel of
// *BlockTransaction or error.
func (e *BlockConverter) Rebase(from, to int64, firstNForcedResults []*BlockTransaction) (<- chan interface{}, error) {
	return nil, nil
}

