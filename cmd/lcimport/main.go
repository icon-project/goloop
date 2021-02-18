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
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/icon-project/goloop/common/crypto"
	"github.com/icon-project/goloop/common/errors"
	"github.com/icon-project/goloop/common/intconv"
	"github.com/icon-project/goloop/common/log"
	"github.com/icon-project/goloop/icon/blockv0"
	"github.com/icon-project/goloop/icon/blockv0/lcstore"
	"github.com/icon-project/goloop/module"
)

const (
	envPrefix = "LOOPCHAIN"
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
	ClearLine = "\x1b[2K"
)

func Statusf(l log.Logger, format string, args ...interface{}) {
	l.Infof(format, args...)
	if l.GetConsoleLevel() < log.InfoLevel {
		fmt.Print(ClearLine)
		fmt.Printf(format, args...)
		fmt.Print("\r")
	}
}

func StatusDone(l log.Logger) {
	if l.GetConsoleLevel() < log.InfoLevel {
		fmt.Print("\n")
	}
}

func newCmdExecuteBlocks(name string, vc *viper.Viper) *cobra.Command {
	cmd := &cobra.Command{
		Args: cobra.RangeArgs(0, 1),
		Use:  name + " [<to>]",
	}
	flags := cmd.PersistentFlags()
	from := flags.Int64("from", -1, "From height(-1 for last)")
	logLevel := flags.String("log_level", "debug", "Default log level")
	consoleLevel := flags.String("console_level", "info", "Console log level")
	logFile := flags.String("log_file", "", "Output logfile")

	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		to := int64(-1)
		if len(args) > 0 {
			if v, err := intconv.ParseInt(args[0], 64); err != nil {
				return errors.Wrapf(err, "InvalidArgument(arg=%s)", args[0])
			} else {
				to = v
			}
		}

		data := vc.GetString("data")

		logger := log.New()
		log.SetGlobalLogger(logger)
		if lv, err := log.ParseLevel(*logLevel); err != nil {
			return errors.Wrapf(err, "InvalidLogLevel(log_level=%s)", *logLevel)
		} else {
			logger.SetLevel(lv)
		}
		if lv, err := log.ParseLevel(*consoleLevel); err != nil {
			return errors.Wrapf(err, "InvalidLogLevel(console_level=%s)", *consoleLevel)
		} else {
			logger.SetConsoleLevel(lv)
		}
		if len(*logFile) > 0 {
			if fw, err := log.NewWriter(&log.WriterConfig{
				Filename: *logFile,
			}); err != nil {
				return err
			} else {
				logger.SetFileWriter(fw)
			}
		}
		for _, l := range logo {
			logger.Infoln(l)
		}

		return executeTransactions(logger, lcDB, data, *from, to)
	}
	return cmd
}

func main() {
	vc := viper.New()
	vc.AutomaticEnv()
	vc.SetEnvPrefix(envPrefix)
	vc.Set("env_prefix", envPrefix)

	root := &cobra.Command{
		Use: os.Args[0],
	}
	pflags := root.PersistentFlags()
	pflags.StringP("store_uri", "b",
		"", "LoopChain Storage URI (leveldb or node endpoint)")
	pflags.StringP("data", "d",
		".chain/import", "Data path to store node data")
	if err := vc.BindPFlags(pflags); err != nil {
		log.Errorf("Fail to bind flags err=%+v", err)
		os.Exit(1)
	}

	root.AddCommand(newCmdGetTx("tx"))
	root.AddCommand(newCmdGetResult("result"))
	root.AddCommand(newCmdGetBlock("block"))
	root.AddCommand(newCmdVerifyBlock("verify"))
	root.AddCommand(newCmdExecuteBlocks("execute", vc))

	root.SilenceUsage = true
	root.PersistentPreRunE = func(cmd *cobra.Command, args []string) error {
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
