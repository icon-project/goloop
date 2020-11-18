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
	"github.com/icon-project/goloop/common/db"
	cs "github.com/icon-project/goloop/consensus"
)

var dbPath string
var walPath string
var dbType string

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
	curHeight, err := block.GetLastHeight(d)
	fmt.Printf("Current block height : %v \n", curHeight)
	fmt.Printf("Target block height  : %v \n", height)
	fmt.Printf("Confirm reset? (y/n) ")
	reader := bufio.NewReader(os.Stdin)
	confirm, err := reader.ReadString('\n')
	if err != nil {
		return nil
	}
	confirm = strings.Replace(confirm, "\n", "", -1)
	if confirm != "y" {
		return nil
	}
	err = block.ResetDB(d, height)
	if err != nil {
		return err
	}
	err = cs.ResetWAL(height, walPath)
	if err != nil {
		return err
	}
	return nil
}

func er(msg interface{}) {
	fmt.Println("Error:", msg)
	os.Exit(1)
}

func main() {
	rootCmd := &cobra.Command{
		Use: os.Args[0] + " <height>",
		Args: func(cmd *cobra.Command, args []string) error {
			if len(args)!=1 {
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
	flag.StringVar(&dbType, "db_type", "goleveldb", "Name of database system (badgerdb, goleveldb, boltdb, mapdb)")
	err := rootCmd.Execute()
	if err != nil {
		er(err)
	}
}