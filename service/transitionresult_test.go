/*
 * Copyright 2021 ICON Foundation
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

package service

import (
	"encoding/hex"
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/icon-project/goloop/common/codec"
	"github.com/icon-project/goloop/common/db"
)

func Test_newTransitionResultFromBytes(t *testing.T) {
	s1, _ := hex.DecodeString("6a41c16fb4827945748042f252c39805fb916e3e47f157b3620cfc8ce0c3093d")
	r1, _ := hex.DecodeString("6fa24a70df169c2cb1e10d1ae748096ed0730fc1b1bc869f2ce21abe64f85820")
	r2, _ := hex.DecodeString("ed9e644e59b2ff65446f5f3d7d77c27858facf8aeb3b969470d7499c79f9757c")
	e1, _ := hex.DecodeString("f867a04f820eefa94c3e731d177461f260b90c6f7c71f78170fad578c136a12033b423a0c37eaafab80062deb7eafa82ffc42719604138eef0234da69f789956f7949d1da0bb87db4b20e1d46a2d8f0e6aea6e32fb0451c474a905942086e8e33b3e2b1ab8f800f800")
	b1, _ := hex.DecodeString("a09ec44e59b2ff65426f5f3d7d79c27858f1cf8aeb3b969470d749dc7df97a7e")
	flagZero, _ := hex.DecodeString("00")
	flagBTPData, _ := hex.DecodeString("01")
	flagUnknowns, _ := hex.DecodeString("10")
	type args struct {
		bs []byte
	}
	tests := []struct {
		name    string
		args    args
		want    *transitionResult
		wantErr bool
	}{
		{"NilBytes", args{nil}, &transitionResult{}, false},
		{"EmptyBytes", args{[]byte{}}, &transitionResult{}, false},
		{"OnlyWithStandardHashes", args{codec.BC.MustMarshalToBytes([][]byte{s1, r1, r2})}, &transitionResult{s1, r1, r2, nil, nil}, false},
		{"WithExtensionData", args{codec.BC.MustMarshalToBytes([][]byte{s1, r1, r2, e1})}, &transitionResult{s1, r1, r2, e1, nil}, false},
		{"WithEmptyExFlags", args{codec.BC.MustMarshalToBytes([][]byte{s1, r1, r2, e1, flagZero})}, nil, true},
		{"WithBTPData", args{codec.BC.MustMarshalToBytes([][]byte{s1, r1, r2, e1, flagBTPData, b1})}, &transitionResult{s1, r1, r2, e1, b1}, false},
		{"WithNilBTPData", args{codec.BC.MustMarshalToBytes([][]byte{s1, r1, r2, e1, flagBTPData, nil})}, &transitionResult{s1, r1, r2, e1, nil}, false},
		{"UnknownExFlags", args{codec.BC.MustMarshalToBytes([][]byte{s1, r1, r2, e1, flagUnknowns, b1})}, nil, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := newTransitionResultFromBytes(tt.args.bs)
			if (err != nil) != tt.wantErr {
				t.Errorf("newTransitionResultFromBytes() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("newTransitionResultFromBytes() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_NewBTPContext(t *testing.T) {
	dbase := db.NewMapDB()
	ctx, err := NewBTPContext(dbase, nil)
	assert.NoError(t, err)
	assert.NotNil(t, ctx)
}