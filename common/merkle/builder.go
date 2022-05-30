package merkle

import (
	"container/list"
	"fmt"

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
	OnData(bid db.BucketID, value []byte) error
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

type requestMap map[string]*list.Element

type merkleBuilder struct {
	store      db.Database
	requests   *list.List
	hasherMap  map[string]requestMap
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

func (b *merkleBuilder) OnData(bid db.BucketID, value []byte) error {
	hasher := bid.Hasher()
	if hasher == nil {
		return fmt.Errorf("not found Hasher for bucketID(%s)", bid)
	}

	reqMap := b.hasherMap[hasher.Name()]
	key := hasher.Hash(value)
	reqID := string(key)
	if e, ok := reqMap[reqID]; ok {
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
		delete(reqMap, reqID)
		return nil
	} else {
		return errors.New("IllegalArguments")
	}
}

func (b *merkleBuilder) RequestData(bid db.BucketID, key []byte, requester DataRequester) {
	if key == nil {
		return
	}

	hasher := bid.Hasher()
	if hasher == nil {
		return
	}

	hasherName := hasher.Name()
	if _, present := b.hasherMap[hasherName]; !present {
		b.hasherMap[hasherName] = make(requestMap)
	}

	reqMap := b.hasherMap[hasherName]
	reqID := string(key)
	if e, ok := reqMap[reqID]; ok {
		req := e.Value.(*request)
		req.bucketIDs = append(req.bucketIDs, bid)
		req.requesters = append(req.requesters, requester)
	} else {
		req := &request{
			key:        key,
			bucketIDs:  []db.BucketID{bid},
			requesters: []DataRequester{requester},
		}
		if b.onDataMark != nil {
			e = b.requests.InsertAfter(req, b.onDataMark)
			b.onDataMark = e
		} else {
			e = b.requests.PushBack(req)
		}
		reqMap[reqID] = e
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
		store:     dbase,
		requests:  list.New(),
		hasherMap: make(map[string]requestMap),
	}
	return builder
}
