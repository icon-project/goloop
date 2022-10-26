/*
 * Copyright 2020 ICON Foundation
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

package main

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/go-errors/errors"
	"github.com/spf13/cobra"

	"github.com/icon-project/goloop/block"
	"github.com/icon-project/goloop/common/codec"
	"github.com/icon-project/goloop/common/db"
	cs "github.com/icon-project/goloop/consensus"
	"github.com/icon-project/goloop/module"
)

var dbPath string
var walPath string
var dbType string
var codecType string

func codecForType(t string) codec.Codec {
	if t == "mp" {
		return codec.MP
	}
	return codec.RLP
}

func reset(height int64) error {
	d, err := db.Open(dbPath, dbType, "")
	if err != nil {
		return err
	}
	defer func() {
		if err := d.Close(); err != nil {
			panic(err)
		}
	}()

	cod := codecForType(codecType)
	curHeight, err := block.GetLastHeightWithCodec(d, cod)
	if curHeight <= height {
		return errors.Errorf("invalid target height current=%d target=%d", curHeight, height)
	}
	ver, err := block.GetBlockVersion(d, cod, height)
	if err != nil {
		return err
	}
	if ver <= module.BlockVersion1 {
		return errors.Errorf("unsupported block version=%d height=%d", ver, height)
	}

	fmt.Printf("Current block height : %v \n", curHeight)
	fmt.Printf("Target block height  : %v \n", height)
	fmt.Printf("Confirm reset? (y/n) ")
	reader := bufio.NewReader(os.Stdin)
	confirm, err := reader.ReadString('\n')
	if err != nil {
		return err
	}
	confirm = strings.Replace(confirm, "\n", "", -1)
	if confirm != "y" {
		return nil
	}

	bid, err := block.GetBlockHeaderHashByHeight(d, cod, height)
	if err != nil {
		return err
	}

	cvlBytes, err := block.GetCommitVoteListBytesForHeight(d, cod, height)
	if err != nil {
		return err
	}

	result, err := block.GetBlockResultByHeight(d, cod, height)
	if err != nil {
		return err
	}

	bd, err := block.GetBTPDigestFromResult(d, cod, result)
	if err != nil {
		return err
	}

	vl, err := block.GetNextValidatorsByHeight(d, cod, height)
	if err != nil {
		return err
	}

	vlmBytes, err := cs.WALRecordBytesFromCommitVoteListBytes(
		cvlBytes, height, bid, result, vl, bd, d, cod,
	)
	if err != nil {
		return err
	}

	err = block.ResetDB(d, cod, height)
	if err != nil {
		return err
	}

	err = cs.ResetWAL(height, walPath, vlmBytes)
	if err != nil {
		return err
	}
	return nil
}
func er(msg interface{}) {
	_, _ = fmt.Fprintln(os.Stderr, "Error:", msg)
	os.Exit(1)
}
func main() {
	rootCmd := &cobra.Command{Use: os.Args[0] + " <height>", Args: func(cmd *cobra.Command, args []string) error {
		if len(args) != 1 {
			return errors.New("requires one height argument")
		}
		_, err := strconv.ParseInt(args[0], 0, 64)
		if err != nil {
			return err
		}
		return nil
	},
		Run: func(cmd *cobra.Command, args []string) {
			height, _ := strconv.ParseInt(args[0], 0, 64)
			err := reset(height)
			if err != nil {
				er(err)
			}
		},
	}

	flag := rootCmd.PersistentFlags()
	flag.StringVar(&dbPath, "db_path", "", "DB path. For example, .chain/hxd81df51476cee82617f6fa658ebecc31d24ddce3/bfdc51/db/bfdc51/)")
	flag.StringVar(&walPath, "wal_path", "", "WAL path. For example, .chain/hxd81df51476cee82617f6fa658ebecc31d24ddce3/bfdc51/wal/)")
	flag.StringVar(&dbType, "db_type", "goleveldb",
		fmt.Sprintf("Name of database system (%s)", strings.Join(db.GetSupportedTypes(), ", ")))
	flag.StringVar(&codecType, "codec", "rlp", "Name of data codec (rlp, mp)")
	err := rootCmd.Execute()
	if err != nil {
		er(err)
	}
}
