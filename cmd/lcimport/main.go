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

package main

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"
	"text/scanner"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/icon-project/goloop/common"
	"github.com/icon-project/goloop/common/codec"
	"github.com/icon-project/goloop/common/containerdb"
	"github.com/icon-project/goloop/common/crypto"
	"github.com/icon-project/goloop/common/errors"
	"github.com/icon-project/goloop/common/intconv"
	"github.com/icon-project/goloop/common/log"
	"github.com/icon-project/goloop/icon/blockv0"
	"github.com/icon-project/goloop/icon/blockv0/lcstore"
	"github.com/icon-project/goloop/module"
	"github.com/icon-project/goloop/service/scoredb"
	"github.com/icon-project/goloop/service/state"
)

const (
	envPrefix = "LCIMPORT"
)

const (
	vcKeyExecutor = "internal.executor"
)

var lcDB *lcstore.Store

func newCmdGetTx(name string) *cobra.Command {
	return &cobra.Command{
		Use: name + " <tid> ...",
		RunE: func(cmd *cobra.Command, args []string) error {
			for _, arg := range args {
				if strings.HasPrefix(arg, "0x") {
					arg = arg[2:]
				}
				bs, err := hex.DecodeString(arg)
				if err != nil {
					return err
				}
				info, err := lcDB.GetTransactionInfoByTransaction(bs)
				if err != nil {
					return err
				}
				jso, err := info.Transaction.ToJSON(module.JSONVersionLast)
				if err != nil {
					return err
				}
				txs, err := json.MarshalIndent(jso, "", "  ")
				if err != nil {
					return err
				}
				fmt.Println(string(txs))
			}
			return nil
		},
	}
}

func newCmdGetResult(name string) *cobra.Command {
	return &cobra.Command{
		Use: name + " <tid> ...",
		RunE: func(cmd *cobra.Command, args []string) error {
			for _, arg := range args {
				if strings.HasPrefix(arg, "0x") {
					arg = arg[2:]
				}
				bs, err := hex.DecodeString(arg)
				if err != nil {
					return err
				}
				info, err := lcDB.GetTransactionInfoJSONByTransaction(bs)
				if err != nil {
					return err
				}
				info, err = json.MarshalIndent(json.RawMessage(info), "", "  ")
				if err != nil {
					return err
				}
				fmt.Println(string(info))
			}
			return nil
		},
	}
}

func showBlock(block []byte) {
	if bs, err := json.MarshalIndent(json.RawMessage(block), "", "  "); err == nil {
		block = bs
	}
	fmt.Println(string(block))
	blk, err := blockv0.ParseBlock(block, lcDB)
	if err != nil {
		log.Warnf("[!] Fail to parse block err=%+v", err)
	} else {
		if err := blk.Verify(nil); err != nil {
			log.Warnf("[!] Fail to verify block err=%+v", err)
		}
	}
}

func newCmdGetBlock(name string) *cobra.Command {
	return &cobra.Command{
		Use: name + " <height or id> ...",
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 {
				block, err := lcDB.GetLastBlockJSON()
				if err != nil {
					return err
				}
				showBlock(block)
			}
			for _, arg := range args {
				var block []byte
				if len(arg) >= crypto.HashLen {
					if strings.HasPrefix(arg, "0x") {
						arg = arg[2:]
					}
					bs, err := hex.DecodeString(arg)
					if err != nil {
						return errors.UnsupportedError.Errorf("Not supported id=%#x", bs)
					}
					block, err = lcDB.GetBlockJSONByID(bs)
					if err != nil {
						return err
					}
				} else {
					height, err := intconv.ParseInt(arg, 64)
					if err != nil {
						return err
					}
					block, err = lcDB.GetBlockJSONByHeight(int(height))
					if err != nil {
						return err
					}
				}
				showBlock(block)
			}
			return nil
		},
	}
}

func newCmdVerifyBlock(name string) *cobra.Command {
	return &cobra.Command{
		Args: cobra.RangeArgs(1, 2),
		Use:  name + " <from> [<to>]",
		RunE: func(cmd *cobra.Command, args []string) error {
			from, err := intconv.ParseInt(args[0], 64)
			if err != nil {
				return err
			}
			var to int64 = -1
			if len(args) > 1 {
				to, err = intconv.ParseInt(args[1], 64)
				if err != nil {
					return err
				}
			}
			var prev blockv0.Block
			for idx := from; to == -1 || idx <= to; idx = idx + 1 {
				fmt.Fprintf(os.Stderr, "\r\x1b[2k[#] Block[%12d]..", idx)
				blkJSON, err := lcDB.GetBlockJSONByHeight(int(idx))
				if err != nil {
					return err
				}
				blk, err := blockv0.ParseBlock(blkJSON, lcDB)
				if err != nil {
					if js, err := json.MarshalIndent(json.RawMessage(blkJSON), "", "  "); err == nil {
						blkJSON = js
					}
					fmt.Fprintf(os.Stderr, "DECODE_FAIL %+v\n", err)
					fmt.Fprintln(os.Stdout, string(blkJSON))
					return err
				}
				if err := blk.Verify(prev); err != nil {
					if js, err := json.MarshalIndent(json.RawMessage(blkJSON), "", "  "); err == nil {
						blkJSON = js
					}
					fmt.Fprintf(os.Stderr, "VERIFY_FAIL %+v\n", err)
					fmt.Fprintln(os.Stdout, string(blkJSON))
					return err
				}
				prev = blk
			}
			fmt.Fprint(os.Stderr, "\n")
			return nil
		},
	}
}

var logo = []string{
	"  _____ _____ ____  _   _ ___  ",
	" |_   _/ ____/ __ \\| \\ | |__ \\ ",
	"   | || |   | |  | |  \\| |  ) |",
	"   | || |   | |  | | . ` | / / ",
	"  _| || |___| |__| | |\\  |/ /_ ",
	" |_____\\_____\\____/|_| \\_|____| IMPORTER",
}

const (
	CursorUp  = "\x1b[1A"
	ClearLine = "\x1b[2K"
)

var statusDisplay bool

func Statusf(l log.Logger, format string, args ...interface{}) {
	l.Infof(format, args...)
	if l.GetConsoleLevel() < log.InfoLevel {
		if statusDisplay {
			fmt.Print(CursorUp + ClearLine)
		}
		fmt.Printf(format, args...)
		fmt.Print("\n")
		statusDisplay = true
	}
}

func newCmdExecutor(parent *cobra.Command, name string, vc *viper.Viper) *cobra.Command {
	cmd := &cobra.Command{
		Use: name,
	}
	cmd.PersistentPreRunE = func(cmd *cobra.Command, args []string) error {
		if err := parent.PersistentPreRunE(cmd, args); err != nil {
			return err
		}
		if executor, err := NewExecutor(log.GlobalLogger(), lcDB, vc.GetString("data")); err != nil {
			return err
		} else {
			vc.Set(vcKeyExecutor, executor)
		}
		return nil
	}
	cmd.AddCommand(newCmdExecuteBlocks(cmd, "run", vc))
	cmd.AddCommand(newCmdLastHeight(cmd, "last", vc))
	cmd.AddCommand(newCmdState(cmd, "state", vc))
	return cmd
}

func parseParams(p string) ([]string, error) {
	var params []string
	s := new(scanner.Scanner)
	s.Init(bytes.NewBufferString(p))
	s.Mode = scanner.ScanIdents | scanner.ScanStrings | scanner.ScanInts
	for {
		switch value := s.Scan(); value {
		case scanner.EOF:
			return params, nil
		case scanner.Ident, scanner.Int:
			params = append(params, s.TokenText())
		case scanner.String, scanner.RawString:
			token := s.TokenText()
			var str string
			if err := json.Unmarshal([]byte(token), &str); err != nil {
				return nil, errors.IllegalArgumentError.Wrapf(err,
					"Invalid String(%q)", token)
			}
			params = append(params, str)
		case '-':
			if s.Scan() == scanner.Int {
				params = append(params, "-"+s.TokenText())
			} else {
				return nil, errors.IllegalArgumentError.Errorf("InvalidTokenAfterMinus")
			}
		case '.':
		default:
			return nil, errors.IllegalArgumentError.Errorf(
				"Unknown character=%c", value)
		}
	}
}

func toKeys(params []string) []interface{} {
	var keys []interface{}
	for _, p := range params {
		if v, err := strconv.ParseInt(p, 0, 64); err == nil {
			keys = append(keys, v)
		} else if addr, err := common.NewAddressFromString(p); err == nil {
			keys = append(keys, addr)
		} else {
			keys = append(keys, p)
		}
	}
	return keys
}

func showValue(value containerdb.Value, ts string) {
	switch ts {
	case "int":
		fmt.Printf("%d\n", value.BigInt())
	case "bool":
		fmt.Printf("%v\n", value.Bool())
	case "str", "string":
		fmt.Printf("%q\n", value.String())
	case "addr", "Address":
		fmt.Printf("%s\n", value.Address().String())
	case "bytes":
		fmt.Printf("%#x\n", value.Bytes())
	default:
		log.Warnf("Unknown type=%s bytes=%#x", ts, value.Bytes())
	}
}

func showAccount(addr module.Address, ass state.AccountSnapshot, params []string) error {
	if len(params) == 0 {
		fmt.Printf("Account[%s]\n", addr.String())
		fmt.Printf("- Balance : %#d\n", ass.GetBalance())
		if ass.IsContract() {
			fmt.Printf("- Owner   : %s\n", ass.ContractOwner())
			api, err := ass.APIInfo()
			if err != nil {
				return err
			}
			apijs, _ := JSONMarshalIndent(api)
			fmt.Printf("- API Info\n%s\n", apijs)
		}
		return nil
	} else {
		if len(params) < 3 {
			return errors.Errorf("InvalidArguments(%+v)", params)
		}
		prefix := params[0]
		params = params[1:]
		store := containerdb.NewBytesStoreStateWithSnapshot(ass)
		_, _ = store, prefix
		switch prefix {
		case "var":
			suffix := params[len(params)-1]
			keys := toKeys(params[:len(params)-1])
			vardb := scoredb.NewVarDB(store, keys...)
			showValue(vardb, suffix)
			return nil
		case "array":
			suffix := params[len(params)-1]
			params = params[:len(params)-1]
			var keys []interface{}
			if suffix != "size" {
				if len(params) < 2 {
					return errors.IllegalArgumentError.New("")
				}
				idxStr := params[len(params)-1]
				keys = toKeys(params[:len(params)-1])
				idx, err := strconv.ParseInt(idxStr, 0, 64)
				if err != nil {
					return errors.IllegalArgumentError.Wrapf(err,
						"InvalidArrayIndex(value=%s)", idxStr)
				}
				arraydb := scoredb.NewArrayDB(store, keys...)
				value := arraydb.Get(int(idx))
				showValue(value, suffix)
				return nil
			} else {
				keys := toKeys(params)
				arraydb := scoredb.NewArrayDB(store, keys...)
				fmt.Printf("%d\n", arraydb.Size())
				return nil
			}
		case "dict":
			suffix := params[len(params)-1]
			params = params[:len(params)-1]
			name := params[0]
			keys := toKeys(params[1:])

			dictdb := scoredb.NewDictDB(store, name, len(keys))
			value := dictdb.Get(keys...)
			if value == nil {
				fmt.Println("nil")
			}
			showValue(value, suffix)
		default:
			return errors.IllegalArgumentError.Errorf(
				"InvalidPrefix(prefix=%s)", prefix)
		}
		return nil
	}
}

func showWorld(wss state.WorldSnapshot, params []string) error {
	if len(params) < 1 {
		return errors.IllegalArgumentError.New("" +
			"Address need to be specified")
	}
	addr := common.MustNewAddressFromString(params[0])
	if addr == nil {
		return errors.IllegalArgumentError.Errorf(
			"InvalidAddress(addr=%s)", params[0])
	}
	params = params[1:]
	ass := wss.GetAccountSnapshot(addr.ID())
	return showAccount(addr, ass, params)
}

func newCmdState(parent *cobra.Command, name string, vc *viper.Viper) *cobra.Command {
	cmd := &cobra.Command{
		Args: cobra.RangeArgs(0, 1),
		Use:  name + " [<expr>]",
	}
	pflags := cmd.PersistentFlags()
	pHeight := pflags.Int64("height", 0, "Height of the state (0 for last height)")
	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		ex := vc.Get(vcKeyExecutor).(*Executor)
		height := *pHeight
		if height == 0 {
			height = ex.getLastHeight()
		}

		if len(args) >= 1 {
			wss, err := ex.NewWorldSnapshot(height)
			if err != nil {
				return err
			}
			for _, arg := range args {
				params, err := parseParams(arg)
				if err != nil {
					return err
				}
				if err := showWorld(wss, params); err != nil {
					return err
				}
			}
		} else {
			blk, err := ex.GetBlockByHeight(height)
			if err != nil {
				return err
			}
			fmt.Printf("Block[%d] - %#x\n", height, blk.ID())
			var values [][]byte
			result := blk.Result()
			if len(result) > 0 {
				if _, err := codec.BC.UnmarshalFromBytes(result, &values); err != nil {
					return err
				}
				fmt.Printf("- World State Hash  : %#x\n", values[0])
				fmt.Printf("- Patch Result Hash : %#x\n", values[1])
				fmt.Printf("- Normal Result Hash: %#x\n", values[2])
				if len(values) >= 3 {
					fmt.Printf("- Extension Data    : %#x\n", values[3])
				}
			}
			fmt.Printf("- Total Transactions: %d", blk.TxTotal())
		}
		return nil
	}
	return cmd
}

func newCmdLastHeight(parent *cobra.Command, name string, vc *viper.Viper) *cobra.Command {
	cmd := &cobra.Command{
		Args: cobra.NoArgs,
		Use:  name,
		RunE: func(cmd *cobra.Command, args []string) error {
			if executor, ok := vc.Get(vcKeyExecutor).(*Executor); ok {
				fmt.Println(executor.getLastHeight())
				return nil
			} else {
				return errors.New("NoValidExecutor")
			}
		},
	}
	return cmd
}

func newCmdExecuteBlocks(parent *cobra.Command, name string, vc *viper.Viper) *cobra.Command {
	cmd := &cobra.Command{
		Args: cobra.RangeArgs(0, 1),
		Use:  name + " [<to>]",
	}
	flags := cmd.PersistentFlags()
	from := flags.Int64("from", -1, "From height(-1 for last)")
	noCache := flags.Bool("no_cache", false, "Disable cache")

	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		to := int64(-1)
		if len(args) > 0 {
			if v, err := intconv.ParseInt(args[0], 64); err != nil {
				return errors.Wrapf(err, "InvalidArgument(arg=%s)", args[0])
			} else {
				to = v
			}
		}
		for _, l := range logo {
			log.Infoln(l)
		}
		executor := vc.Get(vcKeyExecutor).(*Executor)
		return executor.Execute(*from, to, *noCache)
	}
	return cmd
}

func main() {
	vc := viper.New()
	vc.AutomaticEnv()
	vc.SetEnvPrefix(envPrefix)

	root := &cobra.Command{
		Use: os.Args[0],
	}
	pflags := root.PersistentFlags()
	pflags.StringP("store_uri", "b",
		"", "LoopChain Storage URI (leveldb or node endpoint)")
	pflags.StringP("data", "d",
		".chain/import", "Data path to store node data")
	pflags.String("log_level", "debug", "Default log level")
	pflags.String("console_level", "info", "Console log level")
	pflags.String("log_file", "", "Output logfile")
	if err := vc.BindPFlags(pflags); err != nil {
		log.Errorf("Fail to bind flags err=%+v", err)
		os.Exit(1)
	}

	root.AddCommand(newCmdGetTx("tx"))
	root.AddCommand(newCmdGetResult("result"))
	root.AddCommand(newCmdGetBlock("block"))
	root.AddCommand(newCmdVerifyBlock("verify"))
	root.AddCommand(newCmdExecutor(root, "executor", vc))

	root.SilenceUsage = true
	root.PersistentPreRunE = func(cmd *cobra.Command, args []string) error {
		logger := log.GlobalLogger()
		logLevel := vc.GetString("log_level")
		if lv, err := log.ParseLevel(logLevel); err != nil {
			return errors.Wrapf(err, "InvalidLogLevel(log_level=%s)", logLevel)
		} else {
			logger.SetLevel(lv)
		}
		consoleLevel := vc.GetString("console_level")
		if lv, err := log.ParseLevel(consoleLevel); err != nil {
			return errors.Wrapf(err, "InvalidLogLevel(console_level=%s)", consoleLevel)
		} else {
			logger.SetConsoleLevel(lv)
		}
		logFile := vc.GetString("log_file")
		if len(logFile) > 0 {
			if fw, err := log.NewWriter(&log.WriterConfig{
				Filename: logFile,
			}); err != nil {
				return err
			} else {
				logger.SetFileWriter(fw)
			}
		}
		uri := vc.GetString("store_uri")
		if db, err := lcstore.OpenStore(uri); err != nil {
			return errors.Wrapf(err, "OpenFailure(uri=%s)", uri)
		} else {
			lcDB = db
		}
		return nil
	}

	if err := root.Execute(); err != nil {
		log.Errorf("Fail to execute err=%+v", err)
		os.Exit(1)
	}
}
