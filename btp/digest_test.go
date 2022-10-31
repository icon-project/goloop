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

package btp

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/icon-project/goloop/common/db"
	"github.com/icon-project/goloop/module"
)

func TestDigest_ZeroValueDigest(t *testing.T) {
	// use only view
	s := newComplexTestBuilderSetup(t)

	assert := assert.New(t)
	bd, err := NewDigestFromBytes(nil)
	assert.NoError(err)
	dbase := db.NewMapDB()
	bs, err := NewSection(bd, s.view, dbase)
	assert.NoError(err)
	assert.EqualValues(0, len(bs.NetworkTypeSections()))
	nts, err := bs.NetworkTypeSectionFor(0)
	assert.Nil(nts)
	assert.Error(err)
}

func TestDigest_EmptyDigest(t *testing.T) {
	assert := assert.New(t)
	bd, err := NewDigestFromBytes(nil)
	assert.NoError(err)
	assert.EqualValues([]byte(nil), bd.Bytes())
	assert.EqualValues([]byte(nil), bd.Hash())
}

func assertEqualDigest(t *testing.T, d1 module.BTPDigest, d2 module.BTPDigest) {
	assert.Equal(t, len(d1.NetworkTypeDigests()), len(d2.NetworkTypeDigests()))
	for i := 0; i < len(d1.NetworkTypeDigests()); i++ {
		assertEqualNetworkTypeDigest(t, d1.NetworkTypeDigests()[i], d2.NetworkTypeDigests()[i])
	}
}

func assertEqualNetworkTypeDigest(t *testing.T, ntd1 module.NetworkTypeDigest, ntd2 module.NetworkTypeDigest) {
	assert.Equal(t, ntd1.NetworkTypeID(), ntd2.NetworkTypeID())
	assert.EqualValues(t, ntd1.NetworkTypeSectionHash(), ntd2.NetworkTypeSectionHash())
	assert.Equal(t, len(ntd1.NetworkDigests()), len(ntd2.NetworkDigests()))
	for i := 0; i < len(ntd1.NetworkDigests()); i++ {
		assertEqualNetworkDigest(t, ntd1.NetworkDigests()[i], ntd2.NetworkDigests()[i])
	}
}

func assertEqualNetworkDigest(t *testing.T, nd1 module.NetworkDigest, nd2 module.NetworkDigest) {
	assert.Equal(t, nd1.NetworkID(), nd2.NetworkID())
	assert.Equal(t, nd1.NetworkSectionHash(), nd2.NetworkSectionHash())
	assert.Equal(t, nd1.MessagesRoot(), nd2.MessagesRoot())
}

func TestDigest_FlushAndFromBytes(t *testing.T) {
	assert := assert.New(t)
	s := newComplexTestBuilderSetup(t)

	d := s.bs.Digest()
	mdb := db.NewMapDB()
	err := d.Flush(mdb)
	assert.NoError(err)

	bk, _ := mdb.GetBucket(db.BytesByHash)
	digestBytes, err := bk.Get(d.Hash())
	assert.NoError(err)
	d2, err := NewDigestFromBytes(digestBytes)
	assert.NoError(err)
	assert.EqualValues(d2.Bytes(), d.Bytes())
	assertEqualDigest(t, d2, d)

	ml1, err := d2.NetworkTypeDigestFor(1).NetworkDigestFor(1).MessageList(mdb, s.mod)
	assert.NoError(err)
	assert.EqualValues(1, ml1.Len())
	m1, err := ml1.Get(0)
	assert.NoError(err)
	assert.EqualValues("a", m1.Bytes())

}

func TestDigest_Sections(t *testing.T) {
	assert := assert.New(t)
	s := newComplexTestBuilderSetup(t)
	s.updateView()

	d := s.bs.Digest()
	mdb := db.NewMapDB()
	err := d.Flush(mdb)
	assert.NoError(err)

	bk, _ := mdb.GetBucket(db.BytesByHash)
	digestBytes, err := bk.Get(d.Hash())
	assert.NoError(err)
	d2, err := NewDigestFromBytes(digestBytes)
	assert.NoError(err)
	assert.EqualValues(d2.Bytes(), d.Bytes())

	bs2, _ := NewSection(d2, s.view, mdb)
	nts, _ := s.bs.NetworkTypeSectionFor(1)
	ntsFromBS2, _ := bs2.NetworkTypeSectionFor(1)
	assert.EqualValues(nts.NextProofContext().Bytes(), ntsFromBS2.NextProofContext().Bytes())
	assert.EqualValues(nts.Hash(), ntsFromBS2.Hash())
	ns, _ := nts.NetworkSectionFor(1)
	nsFromBS2, _ := ntsFromBS2.NetworkSectionFor(1)
	assert.EqualValues(ns.Hash(), nsFromBS2.Hash())
}
