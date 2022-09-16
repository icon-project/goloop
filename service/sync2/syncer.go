package sync2

import (
	"context"

	"golang.org/x/sync/errgroup"

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
}

func (s *syncer) setHashes(accountsHash, pReceiptsHash, nReceiptsHash, validatorListHash, extensionData []byte) {
	s.ah = accountsHash
	s.prh = pReceiptsHash
	s.nrh = nReceiptsHash
	s.vlh = validatorListHash
	s.ed = extensionData
}

func (s *syncer) GetBuilder(accountsHash, pReceiptsHash, nReceiptsHash, validatorListHash, extensionData []byte) merkle.Builder {
	s.logger.Debugf("GetBuilder ah(%#x), prh(%#x), nrh(%#x), vlh(%#x), ed(%#x)",
		accountsHash, pReceiptsHash, nReceiptsHash, validatorListHash, extensionData)
	builder := merkle.NewBuilder(s.database)
	ess := s.plt.NewExtensionWithBuilder(builder, extensionData)

	if wss, err := state.NewWorldSnapshotWithBuilder(builder, accountsHash, validatorListHash, ess); err == nil {
		s.wss = wss
	}

	s.prl = txresult.NewReceiptListWithBuilder(builder, pReceiptsHash)
	s.nrl = txresult.NewReceiptListWithBuilder(builder, nReceiptsHash)

	return builder
}

// start Sync
func (s *syncer) SyncWithBuilders(buildersV1 []merkle.Builder, buildersV2 []merkle.Builder) (*Result, error) {
	s.logger.Debugln("SyncWithBuilders")
	egrp, _ := errgroup.WithContext(context.Background())

	for _, builder := range buildersV1 {
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

	for _, builder := range buildersV2 {
		// sync processor with v2 protocol
		sp := newSyncProcessor(builder, reactorsV2, s.logger, false)
		egrp.Go(sp.doSync)
		s.processors = append(s.processors, sp)
	}

	if err := egrp.Wait(); err != nil {
		return nil, err
	}

	result := &Result{
		s.wss, s.prl, s.nrl,
	}
	s.logger.Debugln("SyncWithBuilder done!")
	return result, nil
}

func (s *syncer) ForceSync() (*Result, error) {
	var builders []merkle.Builder

	builder := s.GetBuilder(s.ah, s.prh, s.nrh, s.vlh, s.ed)
	builders = append(builders, builder)

	return s.SyncWithBuilders(builders, nil)
}

// stop Sync
func (s *syncer) Stop() {
	for _, sp := range s.processors {
		sp.Stop()
	}
}

// finalize Sync
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
	ah []byte, prh []byte, nrh []byte, vlh []byte, ed []byte, logger log.Logger, noBuffer bool) Syncer {
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
	}

	return s
}
