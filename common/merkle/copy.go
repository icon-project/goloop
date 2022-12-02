package merkle

import (
	"github.com/icon-project/goloop/common/db"
	"github.com/icon-project/goloop/common/errors"
	"github.com/icon-project/goloop/module"
)

const (
	MaxNumberOfItemsToCopyInRow = 50
)

const (
	DBFlagCopyContext = "copyContext"
)

type CopyContext struct {
	builder Builder
	src     db.Database
	dst     db.Database

	height     int64
	progressCB module.ProgressCallback
}

func (e *CopyContext) Builder() Builder {
	return e.builder
}

func (e *CopyContext) SetProgressCallback(cb module.ProgressCallback) {
	e.progressCB = cb
}

func (e *CopyContext) SetHeight(height int64) {
	e.height = height
}

func (e *CopyContext) reportProgress() error {
	if e.progressCB != nil {
		return e.progressCB(e.height, e.builder.ResolvedCount(), e.builder.UnresolvedCount())
	}
	return nil
}

func (e *CopyContext) Run() error {
	if err := e.reportProgress(); err != nil {
		return err
	}
	for e.builder.UnresolvedCount() > 0 {
		itr := e.builder.Requests()
		processed := 0
		for itr.Next() {
			found := false
			for _, id := range itr.BucketIDs() {
				bk, err := e.src.GetBucket(id)
				if err != nil {
					return err
				}
				v1, err := bk.Get(itr.Key())
				if err != nil {
					return err
				}
				if v1 != nil {
					err := e.builder.OnData(id, v1)
					if err != nil {
						return err
					}
					found = true
					break
				}
			}
			if !found {
				_ = e.reportProgress()
				return errors.NotFoundError.Errorf("FailToFindValue(key=%x)", itr.Key())
			}

			// Prevent massive memory usage by cumulated requests.
			// New requests are inserted before the next, so if it continues
			// to process the next, then the number of requests increases until
			// it reaches the end of the iteration.
			// It could be very big if we sync large tree structure.
			// So, let's stop after process some items.
			if processed += 1; processed >= MaxNumberOfItemsToCopyInRow {
				if err := e.reportProgress(); err != nil {
					return err
				}
				break
			}
		}
	}
	return e.reportProgress()
}

func (e *CopyContext) Copy(id db.BucketID, key []byte) error {
	bk1, err := e.src.GetBucket(id)
	if err != nil {
		return err
	}
	value, err := bk1.Get(key)
	if err != nil {
		return err
	}
	if value == nil {
		return nil
	}
	bk2, err := e.dst.GetBucket(id)
	if err != nil {
		return err
	}
	return bk2.Set(key, value)
}

func (e *CopyContext) Set(id db.BucketID, key, value []byte) error {
	bk, err := e.dst.GetBucket(id)
	if err != nil {
		return err
	}
	return bk.Set(key, value)
}

func (e *CopyContext) SourceDB() db.Database {
	return e.src
}

func (e *CopyContext) TargetDB() db.Database {
	return db.WithFlags(e.dst, db.Flags{
		DBFlagCopyContext: e,
	})
}

func NewCopyContext(src db.Database, dst db.Database) *CopyContext {
	return &CopyContext{
		builder: NewBuilderWithRawDatabase(dst),
		src:     src,
		dst:     dst,
	}
}

// PrepareCopyContext prepares CopyContext for copying src to dst.
// If dst comes from another CopyContext, then it returns the original one
// for tracking progress properly.
func PrepareCopyContext(src db.Database, dst db.Database) *CopyContext {
	if ctx, ok := db.GetFlag(dst, DBFlagCopyContext).(*CopyContext); ok {
		if ctx.src == src {
			return ctx
		}
	}
	return NewCopyContext(src, dst)
}
