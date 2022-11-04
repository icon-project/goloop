package merkle

import (
	"github.com/icon-project/goloop/common/db"
	"github.com/icon-project/goloop/common/errors"
)

const (
	MaxNumberOfItemsToCopyInRow = 50
)

type CopyContext struct {
	builder Builder
	src     db.Database
	dst     db.Database
}

func (e *CopyContext) Builder() Builder {
	return e.builder
}

func (e *CopyContext) Run() error {
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
				return errors.NotFoundError.Errorf("FailToFindValue(key=%x", itr.Key())
			}

			// Prevent massive memory usage by cumulated requests.
			// New requests are inserted before the next, so if it continues
			// to process the next, then the number of requests increases until
			// it reaches the end of the iteration.
			// It could be very big if we sync large tree structure.
			// So, let's stop after process some items.
			if processed += 1; processed >= MaxNumberOfItemsToCopyInRow {
				break
			}
		}
	}
	return nil
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
	return e.dst
}

func NewCopyContext(src db.Database, dst db.Database) *CopyContext {
	return &CopyContext{
		builder: NewBuilderWithRawDatabase(dst),
		src:     src,
		dst:     dst,
	}
}
