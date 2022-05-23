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
	"encoding/binary"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/icon-project/goloop/common/db"
)

func TestDigest_ZeroValueDigest(t *testing.T) {
	assert := assert.New(t)
	bd, err := NewDigestFromBytes(nil)
	assert.NoError(err)
	assert.EqualValues([]byte(nil), bd.Bytes())
	assert.EqualValues([]byte(nil), bd.Hash())
}

func TestDigest_FlushAndFromBytes(t *testing.T) {
	assert := assert.New(t)
	s := newComplexTestBuilderSetup(t)

	d := s.bs.Digest()
	//dumpRLP(t, "  ", d.Bytes())
	mdb := db.NewMapDB()
	err := d.Flush(mdb)
	assert.NoError(err)

	bk, _ := mdb.GetBucket(db.BytesByHash)
	digestBytes, err := bk.Get(d.Hash())
	assert.NoError(err)
	d2, err := NewDigestFromBytes(digestBytes)
	assert.NoError(err)
	assert.EqualValues(d2.Bytes(), d.Bytes())

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
	//dumpRLP(t, "", d.Bytes())
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

func dumpRLP(t *testing.T, indent string, data []byte) {
	p := 0
	for p < len(data) {
		switch q := data[p]; {
		case q < 0x80:
			t.Logf("%sbytes(0x%x:%d) : %x", indent, 1, 1, data[p:p+1])
			p = p + 1
		case q <= 0xb7:
			l := int(q - 0x80)
			t.Logf("%sbytes(0x%x:%d) : %x", indent, l, l, data[p+1:p+1+l])
			p = p + 1 + l
		case q <= 0xbf:
			ll := int(q - 0xb7)
			buf := make([]byte, 8)
			lBytes := data[p+1 : p+1+ll]
			copy(buf[8-ll:], lBytes)
			l := int(binary.BigEndian.Uint64(buf))
			t.Logf("%sbytes(0x%x:%d) : %x", indent, l, l, data[p+1+ll:p+1+ll+l])
			p = p + 1 + ll + l
		case q <= 0xf7:
			l := int(q - 0xc0)
			t.Logf("%slist(0x%x:%d) {", indent, l, l)
			dumpRLP(t, indent+"  ", data[p+1:p+1+l])
			t.Logf("%s}", indent)
			p = p + 1 + l
		case q == 0xf8 && data[p+1] == 0:
			t.Logf("%slist(0x0:0) {} nil?", indent)
			p = p + 2
		default:
			ll := int(q - 0xf7)
			buf := make([]byte, 8)
			lBytes := data[p+1 : p+1+ll]
			copy(buf[8-ll:], lBytes)
			l := int(binary.BigEndian.Uint64(buf))
			t.Logf("%slist(0x%x:%d) {", indent, l, l)
			dumpRLP(t, indent+"  ", data[p+1+ll:p+1+ll+l])
			t.Logf("%s}", indent)
			p = p + 1 + ll + l
		}
	}
}
