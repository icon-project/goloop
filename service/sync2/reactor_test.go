package sync2

import (
	"fmt"

	"github.com/icon-project/goloop/common/db"
	"github.com/icon-project/goloop/common/merkle"
)

type reactorRequestor struct {
	id db.BucketID
}

func (req reactorRequestor) OnData(v []byte, builder merkle.Builder) error {
	fmt.Printf("reactorRequestor bucket id : %v, value : %x\n", req.id, v)
	return nil
}

/*
func TestReactorRequestPack(t *testing.T) {

	srcdb := db.NewMapDB()
	req := &reactorRequestor{
		id: db.BytesByHash,
	}
	// 1 pack has 10 data

	// given pool size 5, request 35
	poolSize := 5
	requestSize := 35
	builder := merkle.NewBuilder(srcdb)
	for i := 0; i < requestSize; i++ {
		builder.RequestData(db.BytesByHash, []byte(fmt.Sprint(i)), req)
	}

	// when get packs
	reqIter := builder.Requests()
	packList := getPacks(reqIter, poolSize)
	packs := packList.([]pack)

	// then expected pack size is 4
	expected1 := 4
	actual1 := len(packs)
	if expected1 != actual1 {
		t.Errorf("PackList size expected : %v, actual : %v", expected1, actual1)
	}

	// given pool size 5, request 51
	newRequestSize := 51
	for i := requestSize; i < newRequestSize; i++ {
		builder.RequestData(db.BytesByHash, []byte(fmt.Sprint(i)), req)
	}

	// when get packs
	reqIter = builder.Requests()
	packList = getPacks(reqIter, poolSize)
	packs = packList.([]pack)

	// then expected pack size is 5
	expected2 := 5
	actual2 := len(packs)
	if expected2 != actual2 {
		t.Errorf("PackList size expected : %v, actual : %v", expected2, actual2)
	}
}

func TestReactorRequestPackV2(t *testing.T) {

	srcdb := db.NewMapDB()
	req := &reactorRequestor{
		id: db.BytesByHash,
	}
	// 1 pack has 10 data

	// given pool size 5, request 35
	poolSize := 5
	requestSize := 35
	builder := merkle.NewBuilder(srcdb)
	for i := 0; i < requestSize; i++ {
		builder.RequestData(db.BytesByHash, []byte(fmt.Sprint(i)), req)
	}

	// when get packsV2
	reqIter := builder.Requests()
	packList := getPacksV2(reqIter, poolSize)
	packs := packList.([]packV2)

	// then expected pack size is 4
	expected1 := 4
	actual1 := len(packs)
	if expected1 != actual1 {
		t.Errorf("PackList size expected : %v, actual : %v", expected1, actual1)
	}

	// given pool size 5, request 51
	newRequestSize := 51
	for i := requestSize; i < newRequestSize; i++ {
		builder.RequestData(db.BytesByHash, []byte(fmt.Sprint(i)), req)
	}

	// when get packsV2
	reqIter = builder.Requests()
	packList = getPacksV2(reqIter, poolSize)
	packs = packList.([]packV2)

	// then expected pack size is 5
	expected2 := 5
	actual2 := len(packs)
	if expected2 != actual2 {
		t.Errorf("PackList size expected : %v, actual : %v", expected2, actual2)
	}
}

func TestReactorDistributePack(t *testing.T) {
	srcdb := db.NewMapDB()
	req := &reactorRequestor{
		id: db.BytesByHash,
	}

	// given pool size 3, request 35 for V1
	poolSizeV1 := 2
	requestSize := 35
	builder := merkle.NewBuilder(srcdb)
	for i := 0; i < requestSize; i++ {
		builder.RequestData(db.BytesByHash, []byte(fmt.Sprint(i)), req)
	}

	// when get packs
	reqIter := builder.Requests()
	packList := getPacks(reqIter, poolSizeV1)
	packs := packList.([]pack)

	// then expected pack size is 2
	expected1 := 2
	actual1 := len(packs)
	if expected1 != actual1 {
		t.Errorf("PackListV1 size expected : %v, actual : %v", expected1, actual1)
	}

	// given pool size 3 for V2
	poolSizeV2 := 3

	// when get packsV2
	packListV2 := getPacksV2(reqIter, poolSizeV2)
	packsV2 := packListV2.([]packV2)

	// then expected pack size is 3
	expected2 := 3
	actual2 := len(packsV2)
	if expected1 != actual1 {
		t.Errorf("PackListV2 size expected : %v, actual : %v", expected2, actual2)
	}
}
*/
