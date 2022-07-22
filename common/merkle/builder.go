package merkle

import (
	"container/list"

	"github.com/icon-project/goloop/common/crypto"
	"github.com/icon-project/goloop/common/db"
	"github.com/icon-project/goloop/common/errors"
)

type DataRequester interface {
	OnData(value []byte, builder Builder) error
}

type RequestIterator interface {
	Next() bool
	Key() []byte
	BucketIDs() []db.BucketID
}

type Builder interface {
	OnData(value []byte) error
	UnresolvedCount() int
	Requests() RequestIterator
	RequestData(id db.BucketID, key []byte, requester DataRequester)
	Database() db.Database
	Flush(write bool) error
}

type request struct {
	key        []byte
	bucketIDs  []db.BucketID
	requesters []DataRequester
}

type merkleBuilder struct {
	store      db.Database
	requests   *list.List
	requestMap map[string]*list.Element
	onDataMark *list.Element
}

type requestIterator struct {
	element *list.Element
	request *request
}

func (i *requestIterator) Next() bool {
	if i.element == nil {
		return false
	} else {
		i.request = i.element.Value.(*request)
		i.element = i.element.Next()
		return true
	}
}

func (i *requestIterator) BucketIDs() []db.BucketID {
	if i.request != nil {
		return i.request.bucketIDs
	}
	return nil
}

func (i *requestIterator) Key() []byte {
	if i.request != nil {
		return i.request.key
	}
	return nil
}

func (b *merkleBuilder) Requests() RequestIterator {
	return &requestIterator{
		element: b.requests.Front(),
	}
}

func (b *merkleBuilder) OnData(value []byte) error {
	key := crypto.SHA3Sum256(value)
	reqID := string(key)
	if e, ok := b.requestMap[reqID]; ok {
		req := e.Value.(*request)
		b.onDataMark = e
		defer func() {
			b.onDataMark = nil
		}()
		for i, requester := range req.requesters {
			bkID := req.bucketIDs[i]
			bk, err := b.store.GetBucket(bkID)
			if err != nil {
				return err
			}
			if err := bk.Set(key, value); err != nil {
				return err
			}
			if err := requester.OnData(value, b); err != nil {
				return err
			}
		}
		b.requests.Remove(e)
		delete(b.requestMap, reqID)
		return nil
	} else {
		return errors.New("IllegalArguments")
	}
}

func (b *merkleBuilder) RequestData(id db.BucketID, key []byte, requester DataRequester) {
	if key == nil {
		return
	}
	reqID := string(key)
	if e, ok := b.requestMap[reqID]; ok {
		req := e.Value.(*request)
		req.bucketIDs = append(req.bucketIDs, id)
		req.requesters = append(req.requesters, requester)
	} else {
		req := &request{
			key:        key,
			bucketIDs:  []db.BucketID{id},
			requesters: []DataRequester{requester},
		}
		if b.onDataMark != nil {
			e = b.requests.InsertAfter(req, b.onDataMark)
			b.onDataMark = e
		} else {
			e = b.requests.PushBack(req)
		}
		b.requestMap[reqID] = e
	}
}

func (b *merkleBuilder) UnresolvedCount() int {
	return b.requests.Len()
}

func (b *merkleBuilder) Flush(write bool) error {
	if ldb, ok := b.store.(db.LayerDB); ok {
		return ldb.Flush(write)
	}
	return nil
}

func (b *merkleBuilder) Database() db.Database {
	return b.store
}

func NewBuilder(dbase db.Database) Builder {
	ldb := db.NewLayerDB(dbase)
	return NewBuilderWithRawDatabase(ldb)
}

func NewBuilderWithRawDatabase(dbase db.Database) Builder {
	builder := &merkleBuilder{
		store:      dbase,
		requests:   list.New(),
		requestMap: make(map[string]*list.Element),
	}
	return builder
}
