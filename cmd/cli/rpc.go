package cli

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"strings"

	"github.com/icon-project/goloop/client"
	"github.com/icon-project/goloop/common"
	"github.com/icon-project/goloop/common/errors"
	"github.com/icon-project/goloop/common/wallet"
	"github.com/icon-project/goloop/module"
	"github.com/icon-project/goloop/server"
	"github.com/icon-project/goloop/server/jsonrpc"
	v3 "github.com/icon-project/goloop/server/v3"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func RpcPersistentPreRunE(vc *viper.Viper, rpcClient *client.ClientV3) func(cmd *cobra.Command, args []string) error {
	return func(cmd *cobra.Command, args []string) error {
		if err := ValidateFlagsWithViper(vc, cmd.Flags()); err != nil {
			return err
		}
		*rpcClient = *client.NewClientV3(vc.GetString("uri"))
		return nil
	}
}

func AddRpcRequiredFlags(c *cobra.Command) {
	pFlags := c.PersistentFlags()
	pFlags.String("uri", "http://127.0.0.1:9080/api/v3", "URI of JSON-RPC API")
	//TODO dump jsonrpc message
	//pFlags.Bool("dump", false, "Print JSON-RPC Request and Response")
	MarkAnnotationCustom(pFlags, "uri")
}

func NewRpcCmd(parentCmd *cobra.Command, parentVc *viper.Viper) (*cobra.Command, *viper.Viper) {
	var rpcClient client.ClientV3
	rootCmd, vc := NewCommand(parentCmd, parentVc, "rpc", "JSON-RPC API")
	rootCmd.PersistentPreRunE = RpcPersistentPreRunE(vc, &rpcClient)
	AddRpcRequiredFlags(rootCmd)
	BindPFlags(vc, rootCmd.PersistentFlags())

	NewSendTxCmd(rootCmd, vc)
	NewMonitorCmd(rootCmd, vc)

	rootCmd.AddCommand(
		&cobra.Command{
			Use:   "lastblock",
			Short: "GetLastBlock",
			Args:  cobra.ExactArgs(0),
			RunE: func(cmd *cobra.Command, args []string) error {
				blk, err := rpcClient.GetLastBlock()
				if err != nil {
					return err
				}
				return JsonPrettyPrintln(os.Stdout, blk)
			},
		},
		&cobra.Command{
			Use:   "blockbyheight HEIGHT",
			Short: "GetBlockByHeight",
			Args:  cobra.ExactArgs(1),
			RunE: func(cmd *cobra.Command, args []string) error {
				height, err := common.ParseInt(args[0], 64)
				if err != nil {
					return err
				}
				param := &v3.BlockHeightParam{Height: jsonrpc.HexInt(common.FormatInt(height))}
				blk, err := rpcClient.GetBlockByHeight(param)
				if err != nil {
					return err
				}
				return JsonPrettyPrintln(os.Stdout, blk)
			},
		},
		&cobra.Command{
			Use:   "blockbyhash HASH",
			Short: "GetBlockByHash",
			Args:  cobra.ExactArgs(1),
			RunE: func(cmd *cobra.Command, args []string) error {
				param := &v3.BlockHashParam{Hash: jsonrpc.HexBytes(args[0])}
				blk, err := rpcClient.GetBlockByHash(param)
				if err != nil {
					return err
				}
				return JsonPrettyPrintln(os.Stdout, blk)
			},
		},
		&cobra.Command{
			Use:   "balance ADDRESS",
			Short: "GetBalance",
			Args:  cobra.ExactArgs(1),
			RunE: func(cmd *cobra.Command, args []string) error {
				param := &v3.AddressParam{Address: jsonrpc.Address(args[0])}
				balance, err := rpcClient.GetBalance(param)
				if err != nil {
					return err
				}
				return JsonPrettyPrintln(os.Stdout, balance)
			},
		},
		&cobra.Command{
			Use:   "scoreapi ADDRESS",
			Short: "GetScoreApi",
			Args:  cobra.ExactArgs(1),
			RunE: func(cmd *cobra.Command, args []string) error {
				param := &v3.ScoreAddressParam{Address: jsonrpc.Address(args[0])}
				scoreApi, err := rpcClient.GetScoreApi(param)
				if err != nil {
					return err
				}
				return JsonPrettyPrintln(os.Stdout, scoreApi)
			},
		},
		&cobra.Command{
			Use:   "totalsupply",
			Short: "GetTotalSupply",
			Args:  cobra.ExactArgs(0),
			RunE: func(cmd *cobra.Command, args []string) error {
				supply, err := rpcClient.GetTotalSupply()
				if err != nil {
					return err
				}
				return JsonPrettyPrintln(os.Stdout, supply)
			},
		},
		&cobra.Command{
			Use:   "txresult HASH",
			Short: "GetTransactionResult",
			Args:  cobra.ExactArgs(1),
			RunE: func(cmd *cobra.Command, args []string) error {
				param := &v3.TransactionHashParam{Hash: jsonrpc.HexBytes(args[0])}
				txResult, err := rpcClient.GetTransactionResult(param)
				if err != nil {
					return err
				}
				return JsonPrettyPrintln(os.Stdout, txResult)
			},
		},
		&cobra.Command{
			Use:   "txbyhash HASH",
			Short: "GetTransactionByHash",
			Args:  cobra.ExactArgs(1),
			RunE: func(cmd *cobra.Command, args []string) error {
				param := &v3.TransactionHashParam{Hash: jsonrpc.HexBytes(args[0])}
				tx, err := rpcClient.GetTransactionByHash(param)
				if err != nil {
					return err
				}
				return JsonPrettyPrintln(os.Stdout, tx)
			},
		})
	callCmd := &cobra.Command{
		Use:   "call",
		Short: "Call",
		Args:  cobra.ExactArgs(0),
		RunE: func(cmd *cobra.Command, args []string) error {
			param := &v3.CallParam{
				FromAddress: jsonrpc.Address(cmd.Flag("from").Value.String()),
				ToAddress:   jsonrpc.Address(cmd.Flag("to").Value.String()),
				DataType:    "call", //refer server/v3/validation.go:27 isCall
			}

			dataM := make(map[string]interface{})
			if dataJson := cmd.Flag("data").Value.String(); dataJson != "" {
				var dataBytes []byte
				if strings.HasPrefix(strings.TrimSpace(dataJson), "{") {
					dataBytes = []byte(dataJson)
				} else {
					var err error
					if dataBytes, err = ioutil.ReadFile(dataJson); err != nil {
						return err
					}
				}
				if err := json.Unmarshal(dataBytes, &dataM); err != nil {
					return err
				}
			}

			if dataMethod := cmd.Flag("data_method").Value.String(); dataMethod != "" {
				dataM["method"] = dataMethod
			}
			if dataParams, err := cmd.Flags().GetStringToString("data_param"); err == nil && len(dataParams) > 0 {
				dataM["params"] = dataParams
			}
			if len(dataM) > 0 {
				param.Data = dataM
			}
			blk, err := rpcClient.Call(param)
			if err != nil {
				return err
			}
			if err = JsonPrettyPrintln(os.Stdout, blk); err != nil {
				return errors.Errorf("failed JsonIntend blk=%+v, err=%+v", blk, err)
			}
			return nil
		},
	}
	rootCmd.AddCommand(callCmd)
	callFlags := callCmd.Flags()
	callFlags.String("from", "", "FromAddress")
	callFlags.String("to", "", "ToAddress")
	callFlags.String("data", "", "Data (JSON string or file)")
	callFlags.String("data_method", "", "Method of Data, will overwrite")
	callFlags.StringToString("data_param", nil,
		"Params of Data, key=value pair, will overwrite")
	MarkAnnotationRequired(callFlags, "to")

	rawCmd := &cobra.Command{
		Use:   "raw FILE",
		Short: "Rpc with raw json file",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			b, err := ioutil.ReadFile(args[0])
			if err != nil {
				return err
			}
			resp, err := rpcClient.Raw(json.RawMessage(b))
			if err != nil {
				return err
			}
			return HttpResponsePrettyPrintln(os.Stdout, resp)
		},
	}
	rootCmd.AddCommand(rawCmd)

	rootCmd.AddCommand(
		&cobra.Command{
			Use:   "databyhash HASH",
			Short: "GetDataByHash",
			Args:  cobra.ExactArgs(1),
			RunE: func(cmd *cobra.Command, args []string) error {
				param := &v3.DataHashParam{Hash: jsonrpc.HexBytes(args[0])}
				raw, err := rpcClient.GetDataByHash(param)
				if err != nil {
					return err
				}
				return JsonPrettyPrintln(os.Stdout, raw)
			},
		},
		&cobra.Command{
			Use:   "blockheaderbyheight HEIGHT",
			Short: "GetBlockHeaderByHeight",
			Args:  cobra.ExactArgs(1),
			RunE: func(cmd *cobra.Command, args []string) error {
				height, err := common.ParseInt(args[0], 64)
				if err != nil {
					return err
				}
				param := &v3.BlockHeightParam{Height: jsonrpc.HexInt(common.FormatInt(height))}
				raw, err := rpcClient.GetBlockHeaderByHeight(param)
				if err != nil {
					return err
				}
				return JsonPrettyPrintln(os.Stdout, raw)
			},
		},
		&cobra.Command{
			Use:   "votesbyheight HEIGHT",
			Short: "GetVotesByHeight",
			Args:  cobra.ExactArgs(1),
			RunE: func(cmd *cobra.Command, args []string) error {
				height, err := common.ParseInt(args[0], 64)
				if err != nil {
					return err
				}
				param := &v3.BlockHeightParam{Height: jsonrpc.HexInt(common.FormatInt(height))}
				raw, err := rpcClient.GetVotesByHeight(param)
				if err != nil {
					return err
				}
				return JsonPrettyPrintln(os.Stdout, raw)
			},
		},
		&cobra.Command{
			Use:   "proofforresult HASH INDEX",
			Short: "GetProofForResult",
			Args:  cobra.ExactArgs(2),
			RunE: func(cmd *cobra.Command, args []string) error {
				idx, err := common.ParseInt(args[1], 64)
				if err != nil {
					return err
				}
				param := &v3.ProofResultParam{
					BlockHash: jsonrpc.HexBytes(args[0]),
					Index:     jsonrpc.HexInt(common.FormatInt(idx)),
				}
				raw, err := rpcClient.GetProofForResult(param)
				if err != nil {
					return err
				}
				return JsonPrettyPrintln(os.Stdout, raw)
			},
		})

	//interactive ()
	return rootCmd, vc
}

func NewSendTxCmd(parentCmd *cobra.Command, parentVc *viper.Viper) *cobra.Command {
	var rpcClient client.ClientV3
	var rpcWallet module.Wallet
	rootCmd, vc := NewCommand(parentCmd, parentVc, "sendtx", "SendTransaction")
	rootCmd.PersistentPreRunE = func(cmd *cobra.Command, args []string) error {
		if err := RpcPersistentPreRunE(vc, &rpcClient)(cmd, args); err != nil {
			return err
		}
		if err := ValidateFlags(cmd.InheritedFlags()); err != nil {
			return err
		}
		var kb, pb []byte
		var err error
		ksf := vc.GetString("key_store")
		if kb, err = ioutil.ReadFile(ksf); err != nil {
			return fmt.Errorf("fail to open KeyStore file=%s err=%+v", ksf, err)
		}
		//key_secret -> key_password
		ksec := vc.GetString("key_secret")
		kpass := vc.GetString("key_password")
		if ksec != "" {
			if pb, err = ioutil.ReadFile(ksec); err != nil {
				return fmt.Errorf("fail to open KeySecret file=%s err=%+v", ksec, err)
			}
		} else if kpass != "" {
			pb = []byte(kpass)
		} else {
			return fmt.Errorf("there is no password information for the KeyStore, use --key_secret or --key_password")
		}
		rpcWallet, err = wallet.NewFromKeyStore(kb, pb)
		if err != nil {
			return fmt.Errorf("fail to create wallet err=%+v", err)
		}
		return nil
	}
	AddRpcRequiredFlags(rootCmd)
	rootPFlags := rootCmd.PersistentFlags()
	rootPFlags.String("key_store", "", "KeyStore file for wallet")
	rootPFlags.String("key_secret", "", "Secret(password) file for KeyStore")
	rootPFlags.String("key_password", "", "Password for the KeyStore file")
	rootPFlags.BytesHex("nid", nil, "Network ID, HexString")
	rootPFlags.Int64("step_limit", 0, "StepLimit")
	MarkAnnotationCustom(rootPFlags, "key_store", "nid", "step_limit")
	BindPFlags(vc, rootCmd.PersistentFlags())

	//fixed protocol
	//rootPFlags.String("version", "", "Version")
	//rootPFlags.String("from", "", "FromAddress")
	//rootPFlags.BytesHex("timestamp", nil, "Timestamp, HexString")
	//rootPFlags.BytesHex("nonce", nil, "Nonce, HexString")
	//rootPFlags.BytesBase64("signature", nil, "Signature of Transaction, HexString")
	//MarkAnnotationHidden(rootPFlags, "version", "from", "timestamp", "nonce", "signature")

	rawCmd := &cobra.Command{
		Use:   "raw FILE",
		Short: "Send transaction with json file",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			b, err := ioutil.ReadFile(args[0])
			if err != nil {
				return err
			}
			param := &v3.TransactionParam{}
			if err := json.Unmarshal(b, param); err != nil {
				return err
			}

			stepLimit := vc.GetInt64("step_limit")
			if stepLimit == 0 {
				param.StepLimit = jsonrpc.HexInt(common.FormatInt(stepLimit))
			}
			strNid := vc.GetString("nid")
			if strNid != "" {
				nid, err := common.ParseInt(strNid, 64)
				if err != nil {
					return err
				}
				param.NetworkID = jsonrpc.HexInt(common.FormatInt(nid))
			}

			txHash, err := rpcClient.SendTransaction(rpcWallet, param)
			if err != nil {
				return err
			}
			return JsonPrettyPrintln(os.Stdout, txHash)
		},
	}
	rootCmd.AddCommand(rawCmd)
	rawFlags := rawCmd.Flags()
	rawFlags.BytesHex("nid", nil, "Network ID, HexString")
	rawFlags.Int64("step_limit", 0, "StepLimit")

	transferCmd := &cobra.Command{
		Use:   "transfer",
		Short: "Coin Transfer Transaction",
		Args:  cobra.ExactArgs(0),
		RunE: func(cmd *cobra.Command, args []string) error {
			val, err := cmd.Flags().GetInt64("value")
			if err != nil {
				return err
			}
			stepLimit := vc.GetInt64("step_limit")
			nid, err := common.ParseInt(vc.GetString("nid"), 64)
			if err != nil {
				return err
			}

			param := &v3.TransactionParam{
				Version:     jsonrpc.HexInt(common.FormatInt(jsonrpc.APIVersion3)),
				FromAddress: jsonrpc.Address(rpcWallet.Address().String()),
				ToAddress:   jsonrpc.Address(cmd.Flag("to").Value.String()),
				Value:       jsonrpc.HexInt(common.FormatInt(val)),
				StepLimit:   jsonrpc.HexInt(common.FormatInt(stepLimit)),
				NetworkID:   jsonrpc.HexInt(common.FormatInt(nid)),
				//Nonce:       "",
			}
			msg, err := cmd.Flags().GetString("message")
			if err != nil {
				return err
			}
			if msg != "" {
				param.DataType = "message"
				param.Data = jsonrpc.HexBytes("0x" + hex.EncodeToString([]byte(msg)))
			}

			txHash, err := rpcClient.SendTransaction(rpcWallet, param)
			if err != nil {
				return err
			}
			return JsonPrettyPrintln(os.Stdout, txHash)
		},
	}
	rootCmd.AddCommand(transferCmd)
	transferFlags := transferCmd.Flags()
	transferFlags.String("to", "", "ToAddress")
	transferFlags.Int64("value", 0, "Value")
	transferFlags.String("message", "", "Message")
	MarkAnnotationRequired(transferFlags, "to", "value")

	callCmd := &cobra.Command{
		Use:   "call",
		Short: "SmartContract Call Transaction",
		Args:  cobra.ExactArgs(0),
		RunE: func(cmd *cobra.Command, args []string) error {
			stepLimit := vc.GetInt64("step_limit")
			nid, err := common.ParseInt(vc.GetString("nid"), 64)
			if err != nil {
				return err
			}

			param := &v3.TransactionParam{
				Version:     jsonrpc.HexInt(common.FormatInt(jsonrpc.APIVersion3)),
				FromAddress: jsonrpc.Address(rpcWallet.Address().String()),
				ToAddress:   jsonrpc.Address(cmd.Flag("to").Value.String()),
				StepLimit:   jsonrpc.HexInt(common.FormatInt(stepLimit)),
				NetworkID:   jsonrpc.HexInt(common.FormatInt(nid)),
				//Nonce:       "",
				DataType: "call",
			}
			dataM := make(map[string]interface{})
			dataM["method"] = cmd.Flag("method").Value.String()
			if dataParams, err := cmd.Flags().GetStringToString("param"); err == nil && len(dataParams) > 0 {
				dataM["params"] = dataParams
			}
			if len(dataM) > 0 {
				param.Data = dataM
			}

			txHash, err := rpcClient.SendTransaction(rpcWallet, param)
			if err != nil {
				return err
			}
			return JsonPrettyPrintln(os.Stdout, txHash)
		},
	}
	rootCmd.AddCommand(callCmd)
	callFlags := callCmd.Flags()
	callFlags.String("to", "", "ToAddress")
	callFlags.String("method", "", "Name of the function to invoke in SCORE")
	callFlags.StringToString("param", nil, "key=value, Function parameters")
	MarkAnnotationRequired(callFlags, "to", "method")

	deployCmd := &cobra.Command{
		Use:   "deploy SCORE_ZIP_FILE",
		Short: "Deploy Transaction",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			stepLimit := vc.GetInt64("step_limit")
			nid, err := common.ParseInt(vc.GetString("nid"), 64)
			if err != nil {
				return err
			}
			param := &v3.TransactionParam{
				Version:     jsonrpc.HexInt(common.FormatInt(jsonrpc.APIVersion3)),
				FromAddress: jsonrpc.Address(rpcWallet.Address().String()),
				ToAddress:   jsonrpc.Address(cmd.Flag("to").Value.String()),
				StepLimit:   jsonrpc.HexInt(common.FormatInt(stepLimit)),
				NetworkID:   jsonrpc.HexInt(common.FormatInt(nid)),
				//Nonce:       "",
				DataType: "deploy",
			}
			if to := cmd.Flag("to").Value.String(); to != "" {
				param.ToAddress = jsonrpc.Address(to)
			}
			dataM := make(map[string]interface{})
			dataM["contentType"] = cmd.Flag("content_type").Value.String()
			isDir, err := IsDirectory(args[0])
			if err != nil {
				return err
			}
			var b []byte
			if isDir {
				if b, err = ZipDirectory(args[0]); err != nil {
					return fmt.Errorf("fail to zip with directory %s err:%+v", args[0], err)
				}
			} else {
				if b, err = ioutil.ReadFile(args[0]); err != nil {
					return fmt.Errorf("fail to read %s err:%+v", args[0], err)
				}
			}
			dataM["content"] = "0x" + hex.EncodeToString(b)
			if dataParams, err := cmd.Flags().GetStringToString("param"); err == nil && len(dataParams) > 0 {
				dataM["params"] = dataParams
			}
			if len(dataM) > 0 {
				param.Data = dataM
			}
			txHash, err := rpcClient.SendTransaction(rpcWallet, param)
			if err != nil {
				return err
			}
			return JsonPrettyPrintln(os.Stdout, txHash)
		},
	}
	rootCmd.AddCommand(deployCmd)
	deployFlags := deployCmd.Flags()
	deployFlags.String("to", "cx0000000000000000000000000000000000000000", "ToAddress")
	deployFlags.String("content_type", "application/zip",
		"Mime-type of the content")
	deployFlags.StringToString("param", nil,
		"key=value, Function parameters will be delivered to on_install() or on_update()")
	MarkAnnotationHidden(deployFlags, "content-type")
	return rootCmd
}

func NewMonitorCmd(parentCmd *cobra.Command, parentVc *viper.Viper) *cobra.Command {
	var rpcClient client.ClientV3
	rootCmd, vc := NewCommand(parentCmd, parentVc, "monitor", "Monitor")
	rootCmd.PersistentPreRunE = RpcPersistentPreRunE(vc, &rpcClient)
	AddRpcRequiredFlags(rootCmd)
	BindPFlags(vc, rootCmd.PersistentFlags())

	monitorBlockCmd := &cobra.Command{
		Use:   "block HEIGHT",
		Short: "MonitorBlock",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			height, err := common.ParseInt(args[0], 64)
			if err != nil {
				return err
			}
			param := &server.BlockRequest{Height: common.HexInt64{Value: height}}
			err = rpcClient.MonitorBlock(param, func(v *server.BlockNotification) {
				JsonPrettyPrintln(os.Stdout, v)
			}, nil)
			if err != nil {
				return err
			}
			return nil
		},
	}
	rootCmd.AddCommand(monitorBlockCmd)

	monitorEventCmd := &cobra.Command{
		Use:   "event HEIGHT",
		Short: "MonitorEvent",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			height, err := common.ParseInt(args[0], 64)
			if err != nil {
				return err
			}
			param := &server.EventRequest{
				Height: common.HexInt64{Value: height},
				Event:  cmd.Flag("event").Value.String(),
			}
			addr := cmd.Flag("addr").Value.String()
			if addr != "" {
				param.Addr = common.NewAddressFromString(addr)
			}

			if datas, err := cmd.Flags().GetStringSlice("data"); err != nil && len(datas) > 0 {
				param.Data = make([]interface{}, len(datas))
				for i, v := range datas {
					param.Data[i] = v
				}
			}
			err = rpcClient.MonitorEvent(param, func(v *server.EventNotification) {
				JsonPrettyPrintln(os.Stdout, v)
			}, nil)
			if err != nil {
				return err
			}
			return nil
		},
	}
	rootCmd.AddCommand(monitorEventCmd)
	monitorEventFlags := monitorEventCmd.Flags()
	monitorEventFlags.String("addr", "", "Addr")
	monitorEventFlags.String("event", "", "Event")
	monitorEventFlags.StringSlice("data", nil, "Data")
	MarkAnnotationRequired(monitorEventFlags, "event")
	//interactive ()
	return rootCmd
}
