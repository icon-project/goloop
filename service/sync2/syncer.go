package sync2

import (
	"context"

	"golang.org/x/sync/errgroup"

	"github.com/icon-project/goloop/btp"
	"github.com/icon-project/goloop/common/db"
	"github.com/icon-project/goloop/common/log"
	"github.com/icon-project/goloop/common/merkle"
	"github.com/icon-project/goloop/module"
	"github.com/icon-project/goloop/service/state"
	"github.com/icon-project/goloop/service/txresult"
)

const (
	protoV1  byte = 1
	protoV2  byte = 2
	protoAny byte = protoV1 | protoV2
)

type syncer struct {
	logger log.Logger

	database   db.Database
	plt        Platform
	reactors   []SyncReactor
	processors []SyncProcessor

	ah  []byte // account hash
	vlh []byte // validator list hash
	ed  []byte // extension data
	prh []byte // patch receipt hash
	nrh []byte // normal receipt hash
	bh  []byte // btp hash

	// Sync Result
	wss state.WorldSnapshot
	prl module.ReceiptList
	nrl module.ReceiptList
	bd  module.BTPDigest
}

func (s *syncer) GetStateBuilder(accountsHash, pReceiptsHash, nReceiptsHash, validatorListHash, extensionData []byte) merkle.Builder {
	s.logger.Debugf("GetStateBuilder ah(%#x), prh(%#x), nrh(%#x), vlh(%#x), ed(%#x)",
		accountsHash, pReceiptsHash, nReceiptsHash, validatorListHash, extensionData)
	builder := merkle.NewBuilder(s.database)
	ess := s.plt.NewExtensionWithBuilder(builder, extensionData)

	if wss, err := state.NewWorldSnapshotWithBuilder(builder, accountsHash, validatorListHash, ess, nil); err == nil {
		s.wss = wss
	}

	s.prl = txresult.NewReceiptListWithBuilder(builder, pReceiptsHash)
	s.nrl = txresult.NewReceiptListWithBuilder(builder, nReceiptsHash)

	return builder
}

func (s *syncer) GetBTPBuilder(btpHash []byte) merkle.Builder {
	s.logger.Debugf("GetBTPBuilder bh(%#x)", btpHash)
	builder := merkle.NewBuilder(s.database)

	btpDigest, err := btp.NewDigestWithBuilder(builder, btpHash)
	if err == nil {
		s.bd = btpDigest
	} else {
		s.logger.Errorf("Failed NewDigestWithBuilder. err(%+v)", err)
		return nil
	}

	return builder
}

// SyncWithBuilders start Sync
func (s *syncer) SyncWithBuilders(stateBuilders []merkle.Builder, btpBuilders []merkle.Builder) (*Result, error) {
	s.logger.Debugln("SyncWithBuilders")
	egrp, _ := errgroup.WithContext(context.Background())

	for _, builder := range stateBuilders {
		// sync processor with v1,v2 protocol
		sp := newSyncProcessor(builder, s.reactors, s.logger, false)
		egrp.Go(sp.doSync)
		s.processors = append(s.processors, sp)
	}

	var reactorsV2 []SyncReactor
	for _, reactor := range s.reactors {
		if reactor.GetVersion() == protoV2 {
			reactorsV2 = append(reactorsV2, reactor)
		}
	}

	for _, builder := range btpBuilders {
		// sync processor with v2 protocol
		sp := newSyncProcessor(builder, reactorsV2, s.logger, false)
		egrp.Go(sp.doSync)
		s.processors = append(s.processors, sp)
	}

	if err := egrp.Wait(); err != nil {
		return nil, err
	}

	var btpData module.BTPDigest
	if s.bd == nil {
		btpData = btp.ZeroDigest
	} else {
		btpData = s.bd
	}

	result := &Result{
		s.wss, s.prl, s.nrl, btpData,
	}
	s.logger.Debugln("SyncWithBuilder done!")
	return result, nil
}

func (s *syncer) ForceSync() (*Result, error) {
	var stateBuilders, btpBuilders []merkle.Builder

	stateBuilder := s.GetStateBuilder(s.ah, s.prh, s.nrh, s.vlh, s.ed)
	stateBuilders = append(stateBuilders, stateBuilder)

	if s.bh != nil {
		btpBuilder := s.GetBTPBuilder(s.bh)
		if btpBuilder != nil {
			btpBuilders = append(btpBuilders, btpBuilder)
		}
	}

	return s.SyncWithBuilders(stateBuilders, btpBuilders)
}

// Stop sync
func (s *syncer) Stop() {
	for _, sp := range s.processors {
		sp.Stop()
	}
}

// Finalize Sync
func (s *syncer) Finalize() error {
	s.logger.Debugf("Finalize :  ah(%#x), prh(%#x), nrh(%#x), vlh(%#x), ed(%#x)\n",
		s.ah, s.prh, s.nrh, s.vlh, s.ed)

	for i, sp := range s.processors {
		sproc := sp.(*syncProcessor)
		if sproc.builder == nil {
			continue
		} else {
			s.logger.Tracef("Flush %v\n", sp)
			if err := sproc.builder.Flush(true); err != nil {
				s.logger.Errorf("Failed to flush for %d builder err(%+v)\n", i, err)
				return err
			}
		}
	}

	s.processors = make([]SyncProcessor, 0)
	return nil
}

func newSyncer(database db.Database, reactors []SyncReactor, plt Platform, logger log.Logger) Syncer {
	s := &syncer{
		logger:   logger,
		database: database,
		reactors: reactors,
		plt:      plt,
	}

	return s
}

func newSyncerWithHashes(database db.Database, reactors []SyncReactor, plt Platform,
	ah, prh, nrh, vlh, ed, bh []byte, logger log.Logger, noBuffer bool) Syncer {
	s := &syncer{
		logger:   logger,
		database: database,
		reactors: reactors,
		plt:      plt,
		ah:       ah,
		vlh:      vlh,
		prh:      prh,
		nrh:      nrh,
		ed:       ed,
		bh:       bh,
	}

	return s
}
