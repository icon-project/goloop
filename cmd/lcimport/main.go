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
	"runtime/pprof"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/icon-project/goloop/cmd/cli"
	"github.com/icon-project/goloop/common/crypto"
	"github.com/icon-project/goloop/common/db"
	"github.com/icon-project/goloop/common/errors"
	"github.com/icon-project/goloop/common/intconv"
	"github.com/icon-project/goloop/common/log"
	"github.com/icon-project/goloop/icon/blockv0"
	"github.com/icon-project/goloop/icon/blockv0/lcstore"
	"github.com/icon-project/goloop/module"
	"github.com/icon-project/goloop/service/txresult"
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
				js, err := lcDB.GetTransactionJSON(bs)
				if err != nil {
					return err
				}
				js, err = json.MarshalIndent(json.RawMessage(js), "", "  ")
				if err != nil {
					return err
				}
				fmt.Println(string(js))
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
				info, err := lcDB.GetResultJSON(bs)
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

func showReps(reps []byte) {
	if bs, err := json.MarshalIndent(json.RawMessage(reps), "", "  "); err == nil {
		reps = bs
	}
	fmt.Println(string(reps))
	rl := new(blockv0.RepsList)
	if err := json.Unmarshal(reps, rl); err != nil {
		log.Warnf("[!] Fail to parse reps err=%+v", err)
		return
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

func newCmdGetReps(name string) *cobra.Command {
	return &cobra.Command{
		Args: cobra.ExactArgs(1),
		Use: name + " <hash>",
		RunE: func(cmd *cobra.Command, args []string) error {
			for _, arg := range args {

				if strings.HasPrefix(arg, "0x") {
					arg = arg[2:]
				}
				bs, err := hex.DecodeString(arg)
				if err != nil {
					return errors.UnsupportedError.Errorf("Not supported id=%#x", bs)
				}

				reps, err := lcDB.GetRepsJSONByHash(bs)
				if err != nil {
					return err
				}
				showReps(reps)
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
			mdb := db.NewMapDB()
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
				if blkV03, ok := blk.(*blockv0.BlockV03); ok {
					txs := blk.NormalTransactions()
					receipts := make([]txresult.Receipt, 0, len(txs))
					for _, tx := range txs {
						jsn, err := lcDB.GetReceiptJSON(tx.ID())
						if err != nil {
							fmt.Fprintf(os.Stderr, "GetReceiptJSON fail %+v\n", err)
							return err
						}
						r, err := txresult.NewReceiptFromJSON(mdb,
							module.NoRevision, jsn)
						if err != nil {
							fmt.Fprintf(os.Stderr, "NewReceiptFromJSON fail %+v\n", err)
							return err
						}
						receipts = append(receipts, r)
					}
					eReceiptsHash := blkV03.ReceiptsHash()
					aReceiptsHash := blockv0.CalcMerkleRootOfReceiptSlice(receipts, txs, blk.Height())
					if !bytes.Equal(eReceiptsHash, aReceiptsHash) {
						for i, tx := range txs {
							jsn, _ := lcDB.GetReceiptJSON(tx.ID())
							fmt.Fprintf(os.Stdout, "receipt[%d] = %s\n", i, jsn)
						}
						return errors.Errorf("ReceiptListHash error (expected=%#x, calc=%#x)", eReceiptsHash, aReceiptsHash)
					}
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

func newCmdExecutor(parent *cobra.Command, name string, vc *viper.Viper) *cobra.Command {
	cmd := &cobra.Command{
		Use: name,
	}
	cmd.PersistentPreRunE = func(cmd *cobra.Command, args []string) error {
		if err := parent.PersistentPreRunE(cmd, args); err != nil {
			return err
		}
		cc := &lcstore.CacheConfig{
			MaxBlocks:  vc.GetInt("max_blocks"),
			MaxWorkers: vc.GetInt("max_workers"),
			MaxRPS:     vc.GetInt("max_rps"),
		}
		logger := log.GlobalLogger()
		fc := lcstore.NewForwardCache(lcDB, logger, cc)
		if executor, err := NewExecutor(logger, fc, vc.GetString("data")); err != nil {
			return err
		} else {
			vc.Set(vcKeyExecutor, executor)
		}
		return nil
	}
	flags := cmd.PersistentFlags()
	flags.Int("max_blocks", 32, "Max number of blocks to cache")
	flags.Int("max_workers", 8, "Max number of workers for cache")
	flags.Int("max_rps", 0, "Max RPS for the server(0:unlimited)")
	vc.BindPFlags(flags)

	cmd.AddCommand(newCmdExecuteBlocks(cmd, "run", vc))
	cmd.AddCommand(newCmdLastHeight(cmd, "last", vc))
	cmd.AddCommand(newCmdState(cmd, "state", vc))
	cmd.AddCommand(newCmdStoredHeight(cmd, "stored", vc))
	cmd.AddCommand(newCmdDownloadBlocks(cmd, "load", vc))
	return cmd
}

func newCmdState(parent *cobra.Command, name string, vc *viper.Viper) *cobra.Command {
	cmd := &cobra.Command{
		Args:  cobra.RangeArgs(0, 1),
		Use:   name + " [<expr>]",
		Short: "Inspect state",
	}
	pflags := cmd.PersistentFlags()
	pHeight := pflags.Int64("height", 0, "Height of the state (0 for last height)")
	// pReadLine := pflags.BoolP("readline", "r", false, "Use command-line for continuous query")
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
			if err :=  showBlockDetail(blk); err != nil {
				return err
			}
			/*
			if *pReadLine {
				wss, err := ex.NewWorldSnapshot(height)
				if err != nil {
					return err
				}
				r, err := readline.New("state> ")
				if err != nil {
					return err
				}
				for {
					line, err := r.Readline()
					if err != nil {
						break
					}
					arg := strings.TrimSpace(line)
					if arg == "" {
						continue
					}
					if arg == "." {
						break
					}
					params, err := parseParams(arg)
					if err != nil {
						fmt.Printf("Error:%+v", err)
						continue
					}
					if err := showWorld(wss, params); err != nil {
						fmt.Printf("Error:%+v", err)
						continue
					}
				}
			}
			*/
		}
		return nil
	}
	return cmd
}

func newCmdLastHeight(parent *cobra.Command, name string, vc *viper.Viper) *cobra.Command {
	cmd := &cobra.Command{
		Args:  cobra.NoArgs,
		Use:   name,
		Short: "Show last finalized block height",
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

var version = "unknown"

func newCmdExecuteBlocks(parent *cobra.Command, name string, vc *viper.Viper) *cobra.Command {
	cmd := &cobra.Command{
		Use:   name + " [<to>]",
		Short: "Execute blocks",
		Args:  cobra.RangeArgs(0, 1),
	}
	flags := cmd.PersistentFlags()
	from := flags.Int64("from", -1, "From height(-1 for last)")
	noStored := flags.Bool("no_stored", false, "No use of stored block data")
	dryRun := flags.Bool("dry_run", false, "Compare stored result)")

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
		log.Infof("Version : %s", version)
		executor := vc.Get(vcKeyExecutor).(*Executor)
		return executor.Execute(*from, to, *noStored, *dryRun)
	}
	return cmd
}

func newCmdDownloadBlocks(parent *cobra.Command, name string, vc *viper.Viper) *cobra.Command {
	cmd := &cobra.Command{
		Use:   name + " [<to>]",
		Short: "Download blocks",
		Args:  cobra.RangeArgs(0, 1),
	}
	flags := cmd.PersistentFlags()
	from := flags.Int64("from", -1, "From height(-1 for last)")
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
		return executor.Download(*from, to)
	}
	return cmd
}

func newCmdStoredHeight(parent *cobra.Command, name string, vc *viper.Viper) *cobra.Command {
	cmd := &cobra.Command{
		Use:   name,
		Short: "Show stored block height",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			if executor, ok := vc.Get(vcKeyExecutor).(*Executor); ok {
				fmt.Println(executor.getStoredHeight())
				return nil
			} else {
				return errors.New("NoValidExecutor")
			}
		},
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
	pflags.String("cpuprofile", "", "CPU Profile")
	pflags.String("memprofile", "", "Memory Profile")
	if err := vc.BindPFlags(pflags); err != nil {
		log.Errorf("Fail to bind flags err=%+v", err)
		os.Exit(1)
	}

	root.AddCommand(newCmdGetTx("tx"))
	root.AddCommand(newCmdGetResult("result"))
	root.AddCommand(newCmdGetBlock("block"))
	root.AddCommand(newCmdVerifyBlock("verify"))
	root.AddCommand(newCmdGetReps("reps"))
	root.AddCommand(newCmdExecutor(root, "executor", vc))

	root.SilenceUsage = true
	root.PersistentPreRunE = func(cmd *cobra.Command, args []string) error {
		if cpuprofile := vc.GetString("cpuprofile"); len(cpuprofile) > 0 {
			cli.StartCPUProfile(cpuprofile)
		}
		if memprofile := vc.GetString("memprofile"); len(memprofile) > 0 {
			cli.StartMemoryProfile(memprofile)
		}
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

	root.PersistentPostRun = func(cmd *cobra.Command, args []string) {
		pprof.StopCPUProfile()
	}

	if err := root.Execute(); err != nil {
		log.Errorf("Fail to execute err=%+v", err)
		os.Exit(1)
	}
}
