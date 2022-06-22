package sync2

import (
	"context"

	"github.com/icon-project/goloop/common/db"
	"github.com/icon-project/goloop/common/log"
	"github.com/icon-project/goloop/common/merkle"
	"github.com/icon-project/goloop/module"
	"github.com/icon-project/goloop/service/state"
	"github.com/icon-project/goloop/service/txresult"
	"golang.org/x/sync/errgroup"
)

const (
	protoV1  byte = 1
	protoV2  byte = 2
	protoAny byte = protoV1 | protoV2
)

type syncer struct {
	log log.Logger

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

func (s *syncer) GetBuilder(accountsHash, pReceiptsHash, nReceiptsHash, validatorListHash, extensionData []byte) merkle.Builder {
	s.log.Debugf("newSyncer ah(%#x), prh(%#x), nrh(%#x), vlh(%#x), ed(%#x)",
		accountsHash, pReceiptsHash, nReceiptsHash, validatorListHash, extensionData)

	s.ah = accountsHash
	s.prh = pReceiptsHash
	s.nrh = nReceiptsHash
	s.vlh = validatorListHash
	s.ed = extensionData

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
	s.log.Debugln("SyncWithBuilders")
	egrp, _ := errgroup.WithContext(context.Background())

	for _, builder := range buildersV1 {
		// sync processor with v1,v2 protocol
		sp := newSyncProcessor(builder, s.reactors, s.log, false)
		egrp.Go(sp.StartSync)
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
		sp := newSyncProcessor(builder, reactorsV2, s.log, false)
		egrp.Go(sp.StartSync)
		s.processors = append(s.processors, sp)
	}

	if err := egrp.Wait(); err != nil {
		return nil, err
	}

	result := &Result{
		s.wss, s.prl, s.nrl,
	}
	s.log.Debugln("SyncWithBuilder done!")
	return result, nil
}

// stop Sync
func (s *syncer) Stop() {
	for _, sp := range s.processors {
		sp.Stop()
	}
}

// finalize Sync
func (s *syncer) Finalize() error {
	s.log.Debugf("Finalize :  ah(%#x), prh(%#x), nrh(%#x), vlh(%#x), ed(%#x)\n",
		s.ah, s.prh, s.nrh, s.vlh, s.ed)

	for i, sp := range s.processors {
		builder := sp.GetBuilder()
		if builder == nil {
			continue
		} else {
			s.log.Tracef("Flush %v\n", sp)
			if err := builder.Flush(true); err != nil {
				s.log.Errorf("Failed to flush for %d builder err(%+v)\n", i, err)
				return err
			}
		}
	}

	s.processors = make([]SyncProcessor, 0)
	return nil
}

func newSyncer(database db.Database, reactors []SyncReactor, plt Platform, log log.Logger) Syncer {
	s := &syncer{
		log:      log,
		database: database,
		reactors: reactors,
		plt:      plt,
	}

	return s
}
