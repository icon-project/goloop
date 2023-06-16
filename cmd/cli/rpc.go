package cli

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"

	"github.com/icon-project/goloop/client"
	"github.com/icon-project/goloop/common"
	"github.com/icon-project/goloop/common/errors"
	"github.com/icon-project/goloop/common/intconv"
	"github.com/icon-project/goloop/common/wallet"
	"github.com/icon-project/goloop/module"
	"github.com/icon-project/goloop/server"
	"github.com/icon-project/goloop/server/jsonrpc"
	v3 "github.com/icon-project/goloop/server/v3"
)

func RpcPersistentPreRunE(vc *viper.Viper, rpcClient *client.ClientV3) func(cmd *cobra.Command, args []string) error {
	return func(cmd *cobra.Command, args []string) error {
		if err := ValidateFlagsWithViper(vc, cmd.Flags()); err != nil {
			return err
		}
		*rpcClient = *client.NewClientV3(vc.GetString("uri"))
		if uri := vc.GetString("debug_uri"); len(uri) > 0 {
			rpcClient.DebugEndPoint = uri
		}
		if vc.GetBool("debug") {
			opts := jsonrpc.IconOptions{}
			opts.SetBool(jsonrpc.IconOptionsDebug, true)
			rpcClient.CustomHeader[jsonrpc.HeaderKeyIconOptions] = opts.ToHeaderValue()
			rpcClient.Pre = func(req *http.Request) error {
				b, err := req.GetBody()
				if err != nil {
					return err
				}
				return JsonPrettyCopyAndClose(os.Stderr, b)
			}
		}
		return nil
	}
}

func readFile(s string) ([]byte, error) {
	if s == "-" {
		return ioutil.ReadAll(os.Stdin)
	} else {
		return ioutil.ReadFile(s)
	}
}

func readJSONObject(s string) (map[string]interface{}, error) {
	if len(s) == 0 {
		return nil, nil
	}
	var bs []byte
	var err error
	if strings.HasPrefix(s, "@") {
		bs, err = ioutil.ReadFile(s[1:])
	} else if strings.HasPrefix(s, "-") {
		bs, err = ioutil.ReadAll(os.Stdin)
	} else {
		bs = []byte(s)
	}
	if err != nil {
		return nil, err
	}
	var params map[string]interface{}
	err = json.Unmarshal(bs, &params)
	if err != nil {
		return nil, err
	} else {
		return params, nil
	}
}

// getParamsFromFlags process "params" string flag and "param" stringToString flag
// for parameter handling.
func getParamsFromFlags(flags *pflag.FlagSet) (interface{}, error){
	var params interface{}
	if pp, err := flags.GetString("params"); err == nil && len(pp) > 0 {
		if parsed, err := readJSONObject(pp) ; err != nil {
			return nil, err
		} else if parsed != nil {
			params = parsed
		}
	}
	if dataParams, err := flags.GetStringToString("param"); err == nil && len(dataParams) > 0 {
		if params != nil {
			pm := params.(map[string]interface{})
			for k, v := range dataParams {
				pm[k] = v
			}
			params = pm
		} else {
			params = dataParams
		}
	}
	return params, nil
}

func AddRpcRequiredFlags(c *cobra.Command) {
	pFlags := c.PersistentFlags()
	pFlags.String("uri", "", "URI of JSON-RPC API")
	pFlags.String("debug_uri", "", "URI of JSON-RPC Debug API")
	pFlags.Bool("debug", false, "JSON-RPC Response with detail information")
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
			Args:  cobra.NoArgs,
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
			Args:  ArgsWithDefaultErrorFunc(cobra.ExactArgs(1)),
			RunE: func(cmd *cobra.Command, args []string) error {
				height, err := intconv.ParseInt(args[0], 64)
				if err != nil {
					return err
				}
				param := &v3.BlockHeightParam{Height: jsonrpc.HexInt(intconv.FormatInt(height))}
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
			Args:  ArgsWithDefaultErrorFunc(cobra.ExactArgs(1)),
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
			Use:   "txresult HASH",
			Short: "GetTransactionResult",
			Args:  ArgsWithDefaultErrorFunc(cobra.ExactArgs(1)),
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
			Args:  ArgsWithDefaultErrorFunc(cobra.ExactArgs(1)),
			RunE: func(cmd *cobra.Command, args []string) error {
				param := &v3.TransactionHashParam{Hash: jsonrpc.HexBytes(args[0])}
				tx, err := rpcClient.GetTransactionByHash(param)
				if err != nil {
					return err
				}
				return JsonPrettyPrintln(os.Stdout, tx)
			},
		})

	balanceCmd := &cobra.Command{
		Use:   "balance ADDRESS",
		Short: "GetBalance",
		Args:  ArgsWithDefaultErrorFunc(cobra.ExactArgs(1)),
		RunE: func(cmd *cobra.Command, args []string) error {
			param := &v3.AddressParam{Address: jsonrpc.Address(args[0])}
			height, err := intconv.ParseInt(cmd.Flag("height").Value.String(), 64)
			if err != nil {
				return err
			}
			if height != -1 {
				param.Height = jsonrpc.HexInt(intconv.FormatInt(height))
			}
			balance, err := rpcClient.GetBalance(param)
			if err != nil {
				return err
			}
			return JsonPrettyPrintln(os.Stdout, balance)
		},
	}
	rootCmd.AddCommand(balanceCmd)
	flags := balanceCmd.Flags()
	flags.Int("height", -1, "BlockHeight")

	scoreAPICmd := &cobra.Command{
		Use:   "scoreapi ADDRESS",
		Short: "GetScoreApi",
		Args:  ArgsWithDefaultErrorFunc(cobra.ExactArgs(1)),
		RunE: func(cmd *cobra.Command, args []string) error {
			param := &v3.ScoreAddressParam{Address: jsonrpc.Address(args[0])}
			height, err := intconv.ParseInt(cmd.Flag("height").Value.String(), 64)
			if err != nil {
				return err
			}
			if height != -1 {
				param.Height = jsonrpc.HexInt(intconv.FormatInt(height))
			}
			scoreApi, err := rpcClient.GetScoreApi(param)
			if err != nil {
				return err
			}
			return JsonPrettyPrintln(os.Stdout, scoreApi)
		},
	}
	rootCmd.AddCommand(scoreAPICmd)
	flags = scoreAPICmd.Flags()
	flags.Int("height", -1, "BlockHeight")

	tsCmd := &cobra.Command{
		Use:   "totalsupply",
		Short: "GetTotalSupply",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			var param *v3.HeightParam
			height, err := intconv.ParseInt(cmd.Flag("height").Value.String(), 64)
			if err != nil {
				return err
			}
			if height != -1 {
				param = &v3.HeightParam{
					Height: jsonrpc.HexInt(intconv.FormatInt(height)),
				}
			}
			supply, err := rpcClient.GetTotalSupply(param)
			if err != nil {
				return err
			}
			return JsonPrettyPrintln(os.Stdout, supply)
		},
	}
	rootCmd.AddCommand(tsCmd)
	flags = tsCmd.Flags()
	flags.Int("height", -1, "BlockHeight")

	callCmd := &cobra.Command{
		Use:   "call",
		Short: "Call",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			param := &v3.CallParam{
				FromAddress: jsonrpc.Address(cmd.Flag("from").Value.String()),
				ToAddress:   jsonrpc.Address(cmd.Flag("to").Value.String()),
				DataType:    "call", //refer server/v3/validation.go:27 isCall
			}
			height, err := intconv.ParseInt(cmd.Flag("height").Value.String(), 64)
			if err != nil {
				return err
			}
			if height != -1 {
				param.Height = jsonrpc.HexInt(intconv.FormatInt(height))
			}

			dataM := make(map[string]interface{})
			if dataJson := cmd.Flag("raw").Value.String(); dataJson != "" {
				var dataBytes []byte
				if strings.HasPrefix(strings.TrimSpace(dataJson), "{") {
					dataBytes = []byte(dataJson)
				} else {
					var err error
					if dataBytes, err = readFile(dataJson); err != nil {
						return err
					}
				}
				if err := json.Unmarshal(dataBytes, &dataM); err != nil {
					return err
				}
			}

			if dataMethod := cmd.Flag("method").Value.String(); dataMethod != "" {
				dataM["method"] = dataMethod
			}
			if dataParams, err := getParamsFromFlags(cmd.Flags()) ; err != nil {
				return err
			} else if dataParams != nil {
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
	callFlags.Int("height", -1, "BlockHeight")
	callFlags.String("method", "",
		"Name of the function to invoke in SCORE, if '--raw' used, will overwrite")
	callFlags.StringToString("params", nil,
		"raw json string or '@<json file>' or '-' for stdin for parameter JSON. it overrides raw one ")
	callFlags.StringToString("param", nil,
		"key=value, Function parameters, if '--raw' used, will overwrite")
	callFlags.String("raw", "", "call with 'data' using raw json file or json-string")
	MarkAnnotationRequired(callFlags, "to")

	rawCmd := &cobra.Command{
		Use:   "raw FILE",
		Short: "Rpc with raw json file",
		Args:  ArgsWithDefaultErrorFunc(cobra.ExactArgs(1)),
		RunE: func(cmd *cobra.Command, args []string) error {
			b, err := readFile(args[0])
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
			Args:  ArgsWithDefaultErrorFunc(cobra.ExactArgs(1)),
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
			Args:  ArgsWithDefaultErrorFunc(cobra.ExactArgs(1)),
			RunE: func(cmd *cobra.Command, args []string) error {
				height, err := intconv.ParseInt(args[0], 64)
				if err != nil {
					return err
				}
				param := &v3.BlockHeightParam{Height: jsonrpc.HexInt(intconv.FormatInt(height))}
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
			Args:  ArgsWithDefaultErrorFunc(cobra.ExactArgs(1)),
			RunE: func(cmd *cobra.Command, args []string) error {
				height, err := intconv.ParseInt(args[0], 64)
				if err != nil {
					return err
				}
				param := &v3.BlockHeightParam{Height: jsonrpc.HexInt(intconv.FormatInt(height))}
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
			Args:  ArgsWithDefaultErrorFunc(cobra.ExactArgs(2)),
			RunE: func(cmd *cobra.Command, args []string) error {
				idx, err := intconv.ParseInt(args[1], 64)
				if err != nil {
					return err
				}
				param := &v3.ProofResultParam{
					BlockHash: jsonrpc.HexBytes(args[0]),
					Index:     jsonrpc.HexInt(intconv.FormatInt(idx)),
				}
				raw, err := rpcClient.GetProofForResult(param)
				if err != nil {
					return err
				}
				return JsonPrettyPrintln(os.Stdout, raw)
			},
		},
		&cobra.Command{
			Use:   "proofforevents BLOCK_HASH TX_INDEX EVENT_INDEXES",
			Short: "GetProofForEvents",
			Args:  ArgsWithDefaultErrorFunc(cobra.ExactArgs(3)),
			RunE: func(cmd *cobra.Command, args []string) error {
				idx, err := intconv.ParseInt(args[1], 64)
				if err != nil {
					return err
				}
				strs := strings.Split(args[2], ",")
				evts := make([]jsonrpc.HexInt, len(strs))
				for i, str := range strs {
					evt, err := intconv.ParseInt(str, 64)
					if err != nil {
						return err
					}
					evts[i] = jsonrpc.HexInt(intconv.FormatInt(evt))
				}
				param := &v3.ProofEventsParam{
					BlockHash: jsonrpc.HexBytes(args[0]),
					Index:     jsonrpc.HexInt(intconv.FormatInt(idx)),
					Events:    evts,
				}
				raw, err := rpcClient.GetProofForEvents(param)
				if err != nil {
					return err
				}
				return JsonPrettyPrintln(os.Stdout, raw)
			},
		})
	scoreStatusCmd := &cobra.Command{
		Use:   "scorestatus ADDRESS",
		Short: "Get status of the smart contract",
		Args:  ArgsWithDefaultErrorFunc(cobra.ExactArgs(1)),
		RunE: func(cmd *cobra.Command, args []string) error {
			param := &v3.ScoreAddressParam{Address: jsonrpc.Address(args[0])}
			height, err := intconv.ParseInt(cmd.Flag("height").Value.String(), 64)
			if err != nil {
				return err
			}
			if height != -1 {
				param.Height = jsonrpc.HexInt(intconv.FormatInt(height))
			}
			scoreStatus, err := rpcClient.GetScoreStatus(param)
			if err != nil {
				return err
			}
			return JsonPrettyPrintln(os.Stdout, scoreStatus)
		},
	}
	rootCmd.AddCommand(scoreStatusCmd)
	flags = scoreStatusCmd.Flags()
	flags.Int("height", -1, "BlockHeight")

	networkInfoCmd := &cobra.Command{
		Use: "networkinfo",
		Short: "Get network info of the endpoint",
		Args: cobra.ExactArgs(0),
		RunE: func(cmd *cobra.Command, args[]string) error {
			info, err := rpcClient.GetNetworkInfo()
			if err != nil {
				return err
			}
			return JsonPrettyPrintln(os.Stdout, info)
		},
	}
	rootCmd.AddCommand(networkInfoCmd)

	rootCmd.AddCommand(
		&cobra.Command{
			Use:   "btpnetwork ID [HEIGHT]",
			Short: "GetBTPNetworkInfo",
			Args:  ArgsWithDefaultErrorFunc(cobra.RangeArgs(1, 2)),
			RunE: func(cmd *cobra.Command, args []string) error {
				p, err := newBTPQueryParam(args)
				if err != nil {
					return err
				}
				r, err := rpcClient.GetBTPNetworkInfo(p)
				if err != nil {
					return err
				}
				return JsonPrettyPrintln(os.Stdout, r)
			},
		},
		&cobra.Command{
			Use:   "btpnetworktype ID [HEIGHT]",
			Short: "GetBTPNetworkTypeInfo",
			Args:  ArgsWithDefaultErrorFunc(cobra.RangeArgs(1, 2)),
			RunE: func(cmd *cobra.Command, args []string) error {
				p, err := newBTPQueryParam(args)
				if err != nil {
					return err
				}
				r, err := rpcClient.GetBTPNetworkTypeInfo(p)
				if err != nil {
					return err
				}
				return JsonPrettyPrintln(os.Stdout, r)
			},
		},
		&cobra.Command{
			Use:   "btpmessages NETWORK_ID HEIGHT",
			Short: "GetBTPMessages",
			Args:  ArgsWithDefaultErrorFunc(cobra.ExactArgs(2)),
			RunE: func(cmd *cobra.Command, args []string) error {
				p, err := newBTPMessagesParam(args)
				if err != nil {
					return err
				}
				r, err := rpcClient.GetBTPMessages(p)
				if err != nil {
					return err
				}
				return JsonPrettyPrintln(os.Stdout, r)
			},
		},
		&cobra.Command{
			Use:   "btpheader NETWORK_ID HEIGHT",
			Short: "GetBTPHeader",
			Args:  ArgsWithDefaultErrorFunc(cobra.ExactArgs(2)),
			RunE: func(cmd *cobra.Command, args []string) error {
				p, err := newBTPMessagesParam(args)
				if err != nil {
					return err
				}
				r, err := rpcClient.GetBTPHeader(p)
				if err != nil {
					return err
				}
				return JsonPrettyPrintln(os.Stdout, r)
			},
		},
		&cobra.Command{
			Use:   "btpproof NETWORK_ID HEIGHT",
			Short: "GetBTPProof",
			Args:  ArgsWithDefaultErrorFunc(cobra.ExactArgs(2)),
			RunE: func(cmd *cobra.Command, args []string) error {
				p, err := newBTPMessagesParam(args)
				if err != nil {
					return err
				}
				r, err := rpcClient.GetBTPProof(p)
				if err != nil {
					return err
				}
				return JsonPrettyPrintln(os.Stdout, r)
			},
		},
		&cobra.Command{
			Use:   "btpsource",
			Short: "GetBTPSourceInformation",
			Args:  ArgsWithDefaultErrorFunc(cobra.ExactArgs(0)),
			RunE: func(cmd *cobra.Command, args []string) error {
				r, err := rpcClient.GetBTPSourceInformation()
				if err != nil {
					return err
				}
				return JsonPrettyPrintln(os.Stdout, r)
			},
		})
	return rootCmd, vc
}

func newHexIntByString(s string) (i jsonrpc.HexInt, err error) {
	var n int64
	if n, err = intconv.ParseInt(s, 64); err != nil {
		return
	}
	i = jsonrpc.HexInt(intconv.FormatInt(n))
	return
}

func newBTPQueryParam(args []string) (p *v3.BTPQueryParam, err error) {
	if len(args) < 1 {
		return nil, fmt.Errorf("invalid args")
	}
	p = &v3.BTPQueryParam{}
	if p.Id, err = newHexIntByString(args[0]); err != nil {
		return nil, err
	}
	if len(args) > 1 {
		if p.Height, err = newHexIntByString(args[1]); err != nil {
			return nil, err
		}
	}
	return p, nil
}

func newBTPMessagesParam(args []string) (p *v3.BTPMessagesParam, err error) {
	if len(args) < 2 {
		return nil, fmt.Errorf("invalid args")
	}
	p = &v3.BTPMessagesParam{}
	if p.NetworkId, err = newHexIntByString(args[0]); err != nil {
		return nil, err
	}
	if p.Height, err = newHexIntByString(args[1]); err != nil {
		return nil, err
	}
	return p, nil
}

func NewSendTxCmd(parentCmd *cobra.Command, parentVc *viper.Viper) *cobra.Command {
	var rpcClient client.ClientV3
	var rpcClientSendTx func(w module.Wallet, params *v3.TransactionParam) (interface{}, error)
	var rpcWallet module.Wallet
	rootCmd, vc := NewCommand(parentCmd, parentVc, "sendtx", "SendTransaction")
	rootCmd.PersistentPreRunE = func(cmd *cobra.Command, args []string) error {
		if err := RpcPersistentPreRunE(vc, &rpcClient)(cmd, args); err != nil {
			return err
		}
		if err := ValidateFlags(cmd.InheritedFlags()); err != nil {
			return err
		}

		if estimate := vc.GetBool("estimate"); estimate {
			rpcClientSendTx = func(w module.Wallet, p *v3.TransactionParam) (interface{}, error) {
				params := &v3.TransactionParamForEstimate{
					Version:     p.Version,
					FromAddress: p.FromAddress,
					ToAddress:   p.ToAddress,
					Value:       p.Value,
					Timestamp:   p.Timestamp,
					NetworkID:   p.NetworkID,
					Nonce:       p.Nonce,
					DataType:    p.DataType,
					Data:        p.Data,
				}
				step, err := rpcClient.EstimateStep(params)
				if err != nil {
					return nil, err
				}
				return step, nil
			}
		} else {
			save := vc.GetString("save")
			rpcClientSendTx = func(w module.Wallet, p *v3.TransactionParam) (interface{}, error) {
				txId, err := rpcClient.SendTransaction(w, p)
				if len(save) > 0 {
					if err := JsonPrettySaveFile(save, 0644, p); err != nil {
						fmt.Fprintf(os.Stderr, "FAIL to save parameter file=%s err=%+v\n", save, err)
					}
				}
				if err != nil {
					return nil, err
				}
				return txId, nil
			}
			if err := CheckFlagsWithViper(vc, cmd.Flags(), "step_limit"); err != nil {
				return err
			}
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
	rootCmd.PersistentPostRunE = func(cmd *cobra.Command, args []string) error {
		txHash, ok := vc.Get("txHash").(*jsonrpc.HexBytes)

		if vc.GetBool("wait") && ok && txHash != nil {
			param := &v3.TransactionHashParam{Hash: *txHash}

			//try waitTransactionResult
			opts := jsonrpc.IconOptions{}
			if vc.GetBool("debug") {
				opts.SetBool(jsonrpc.IconOptionsDebug, true)
			}
			opts.SetInt(jsonrpc.IconOptionsTimeout, vc.GetInt64("wait_timeout")*1000)
			rpcClient.CustomHeader[jsonrpc.HeaderKeyIconOptions] = opts.ToHeaderValue()

			//
			waitInterval := time.Duration(vc.GetInt("wait_interval")) * time.Millisecond
			waitTimeout := time.Duration(vc.GetInt("wait_timeout")) * time.Second
			expireTime := time.Now().Add(waitTimeout)
			toCh := time.After(waitTimeout)
			done := make(chan interface{})
			resultFunc := rpcClient.WaitTransactionResult
			go func() {
				for {
					if txResult, err := resultFunc(param); err != nil {
						je, ok := err.(*jsonrpc.Error)
						if !ok {
							if he, ok := err.(*client.HttpError); ok {
								if he.Response() != "" {
									jre := &jsonrpc.Response{}
									if uErr := json.Unmarshal([]byte(he.Response()), jre); uErr != nil {
										done <- fmt.Errorf("fail to unmarshall jsonrpc.Response err:%+v httpError:%+v", uErr, he)
										return
									}
									je = jre.Error
								}
							}
						}
						if je != nil {
							switch je.Code {
							case jsonrpc.ErrorCodeSystemTimeout:
								wt := expireTime.Sub(time.Now()).Microseconds()
								if wt < 1 {
									done <- fmt.Errorf("timeout %v", waitTimeout)
									return
								}
								opts.SetInt(jsonrpc.IconOptionsTimeout, wt)
								rpcClient.CustomHeader[jsonrpc.HeaderKeyIconOptions] = opts.ToHeaderValue()
								continue
							case jsonrpc.ErrorCodeTimeout:
								done <- je
								return
							case jsonrpc.ErrorCodeMethodNotFound:
								resultFunc = rpcClient.GetTransactionResult
								continue
							case jsonrpc.ErrorCodePending, jsonrpc.ErrorCodeExecuting:
								if time.Now().After(expireTime) {
									done <- fmt.Errorf("timeout %v", waitTimeout)
									return
								}
								time.Sleep(waitInterval)
								continue
							}
						}
					} else {
						done <- txResult
						return
					}
				}
			}()

			select {
			case <-toCh:
				return fmt.Errorf("timeout %v", waitTimeout)
			case v := <-done:
				if err, ok := v.(error); ok {
					return err
				}
				return JsonPrettyPrintln(os.Stdout, v)
			}
		}
		return nil
	}
	AddRpcRequiredFlags(rootCmd)
	rootPFlags := rootCmd.PersistentFlags()
	rootPFlags.String("key_store", "", "KeyStore file for wallet")
	rootPFlags.String("key_secret", "", "Secret(password) file for KeyStore")
	rootPFlags.String("key_password", "", "Password for the KeyStore file")
	rootPFlags.String("nid", "", "Network ID")
	rootPFlags.Int64("step_limit", 0, "StepLimit")
	rootPFlags.Bool("wait", false, "Wait transaction result")
	rootPFlags.Int("wait_interval", 1000, "Polling interval(msec) for wait transaction result")
	rootPFlags.Int("wait_timeout", 10, "Timeout(sec) for wait transaction result")
	rootPFlags.Bool("estimate", false, "Just estimate steps for the tx")
	rootPFlags.String("save", "", "Store transaction to the file")
	MarkAnnotationCustom(rootPFlags, "key_store", "nid")
	BindPFlags(vc, rootCmd.PersistentFlags())
	MarkAnnotationHidden(rootPFlags, "wait", "wait_interval", "wait_timeout")

	//fixed protocol
	//rootPFlags.String("version", "", "Version")
	//rootPFlags.String("from", "", "FromAddress")
	//rootPFlags.BytesHex("timestamp", nil, "Timestamp, HexString")
	//rootPFlags.BytesHex("nonce", nil, "Nonce, HexString")
	//rootPFlags.BytesBase64("signature", nil, "Signature of Transaction, HexString")
	//MarkAnnotationHidden(rootPFlags, "version", "from", "timestamp", "nonce", "signature")

	rawCmd := &cobra.Command{
		Use:   "raw FILE",
		Short: "Send transaction with json file filling nid,version,stepLimit,from and overwriting timestamp and signature",
		Args:  ArgsWithDefaultErrorFunc(cobra.ExactArgs(1)),
		RunE: func(cmd *cobra.Command, args []string) error {
			b, err := readFile(args[0])
			if err != nil {
				return err
			}
			param := &v3.TransactionParam{}
			if err := json.Unmarshal(b, param); err != nil {
				return err
			}
			if param.Version == "" {
				param.Version = v3.VersionValue
			}
			if param.FromAddress == "" {
				param.FromAddress = jsonrpc.Address(rpcWallet.Address().String())
			}
			if param.StepLimit == "" {
				stepLimit := vc.GetInt64("step_limit")
				param.StepLimit = jsonrpc.HexInt(intconv.FormatInt(stepLimit))
			}
			if param.NetworkID == "" {
				strNid := vc.GetString("nid")
				nid, err := intconv.ParseInt(strNid, 64)
				if err != nil {
					return err
				}
				param.NetworkID = jsonrpc.HexInt(intconv.FormatInt(nid))
			}

			txHash, err := rpcClientSendTx(rpcWallet, param)
			if err != nil {
				return err
			}
			vc.Set("txhash", txHash)
			return JsonPrettyPrintln(os.Stdout, txHash)
		},
	}
	rootCmd.AddCommand(rawCmd)

	raw2Cmd := &cobra.Command{
		Use:   "raw2 FILE",
		Short: "Send transaction with json file overwriting timestamp and signature",
		Args:  ArgsWithDefaultErrorFunc(cobra.ExactArgs(1)),
		RunE: func(cmd *cobra.Command, args []string) error {
			b, err := readFile(args[0])
			if err != nil {
				return err
			}
			var param map[string]interface{}
			if err := json.Unmarshal(b, &param); err != nil {
				return err
			}
			txHash, err := rpcClient.SendRawTransaction(rpcWallet, param)
			if err != nil {
				return err
			}
			vc.Set("txhash", txHash)
			return JsonPrettyPrintln(os.Stdout, txHash)
		},
	}
	rootCmd.AddCommand(raw2Cmd)

	raw3Cmd := &cobra.Command{
		Use:   "raw3 FILE",
		Short: "Send transaction with json file",
		Args:  ArgsWithDefaultErrorFunc(cobra.ExactArgs(1)),
		RunE: func(cmd *cobra.Command, args []string) error {
			b, err := readFile(args[0])
			if err != nil {
				return err
			}
			var param json.RawMessage
			if err := json.Unmarshal(b, &param); err != nil {
				return err
			}
			var result jsonrpc.HexBytes
			_, err = rpcClient.Do("icx_sendTransaction", param, &result)
			if err != nil {
				return err
			}
			vc.Set("txhash", &result)
			return JsonPrettyPrintln(os.Stdout, &result)
		},
	}
	rootCmd.AddCommand(raw3Cmd)

	transferCmd := &cobra.Command{
		Use:   "transfer",
		Short: "Coin Transfer Transaction",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			var value common.HexInt
			if _, ok := value.SetString(cmd.Flag("value").Value.String(), 0); !ok {
				return fmt.Errorf("fail to parsing value %s", cmd.Flag("value").Value.String())
			}
			stepLimit := vc.GetInt64("step_limit")
			nid, err := intconv.ParseInt(vc.GetString("nid"), 64)
			if err != nil {
				return err
			}
			param := &v3.TransactionParam{
				Version:     v3.VersionValue,
				FromAddress: jsonrpc.Address(rpcWallet.Address().String()),
				ToAddress:   jsonrpc.Address(cmd.Flag("to").Value.String()),
				Value:       jsonrpc.HexInt(value.String()),
				StepLimit:   jsonrpc.HexInt(intconv.FormatInt(stepLimit)),
				NetworkID:   jsonrpc.HexInt(intconv.FormatInt(nid)),
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

			txHash, err := rpcClientSendTx(rpcWallet, param)
			if err != nil {
				return err
			}
			vc.Set("txhash", txHash)
			return JsonPrettyPrintln(os.Stdout, txHash)
		},
	}
	rootCmd.AddCommand(transferCmd)
	transferFlags := transferCmd.Flags()
	transferFlags.String("to", "", "ToAddress")
	transferFlags.String("value", "", "Value")
	transferFlags.String("message", "", "Message")
	MarkAnnotationRequired(transferFlags, "to", "value")

	callCmd := &cobra.Command{
		Use:   "call",
		Short: "SmartContract Call Transaction",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			stepLimit := vc.GetInt64("step_limit")
			nid, err := intconv.ParseInt(vc.GetString("nid"), 64)
			if err != nil {
				return err
			}

			param := &v3.TransactionParam{
				Version:     v3.VersionValue,
				FromAddress: jsonrpc.Address(rpcWallet.Address().String()),
				ToAddress:   jsonrpc.Address(cmd.Flag("to").Value.String()),
				StepLimit:   jsonrpc.HexInt(intconv.FormatInt(stepLimit)),
				NetworkID:   jsonrpc.HexInt(intconv.FormatInt(nid)),
				//Nonce:       "",
				DataType: "call",
			}
			dataM := make(map[string]interface{})
			if dataJson := cmd.Flag("raw").Value.String(); dataJson != "" {
				var dataBytes []byte
				if strings.HasPrefix(strings.TrimSpace(dataJson), "{") {
					dataBytes = []byte(dataJson)
				} else {
					var err error
					if dataBytes, err = readFile(dataJson); err != nil {
						return err
					}
				}
				if err := json.Unmarshal(dataBytes, &dataM); err != nil {
					return err
				}
			}

			if dataMethod := cmd.Flag("method").Value.String(); dataMethod != "" {
				dataM["method"] = dataMethod
			}
			if dataParams, err := getParamsFromFlags(cmd.Flags()) ; err != nil {
				return err
			} else if dataParams != nil {
				dataM["params"] = dataParams
			}
			if len(dataM) > 0 {
				param.Data = dataM
			}
			if cmd.Flag("value").Value.String() != "" {
				var value common.HexInt
				if _, ok := value.SetString(cmd.Flag("value").Value.String(), 0); !ok {
					return fmt.Errorf("fail to parsing value %s", cmd.Flag("value").Value.String())
				}
				param.Value = jsonrpc.HexInt(value.String())
			}

			txHash, err := rpcClientSendTx(rpcWallet, param)
			if err != nil {
				return err
			}
			vc.Set("txhash", txHash)
			return JsonPrettyPrintln(os.Stdout, txHash)
		},
	}
	rootCmd.AddCommand(callCmd)
	callFlags := callCmd.Flags()
	callFlags.String("to", "", "ToAddress")
	callFlags.String("method", "",
		"Name of the function to invoke in SCORE, if '--raw' used, will overwrite")
	callFlags.String("params", "",
		"raw json string or '@<json file>' or '-' for stdin for parameter JSON, it overrides raw one")
	callFlags.StringToString("param", nil,
		"key=value, Function parameters, if '--raw' used, will overwrite")
	callFlags.String("value", "", "Value of transfer")
	callFlags.String("raw", "", "call with 'data' using raw json file or json-string")
	MarkAnnotationRequired(callFlags, "to", "method")

	deployCmd := &cobra.Command{
		Use:   "deploy SCORE_ZIP_FILE",
		Short: "Deploy Transaction",
		Args:  ArgsWithDefaultErrorFunc(cobra.ExactArgs(1)),
		RunE: func(cmd *cobra.Command, args []string) error {
			stepLimit := vc.GetInt64("step_limit")
			nid, err := intconv.ParseInt(vc.GetString("nid"), 64)
			if err != nil {
				return err
			}
			param := &v3.TransactionParam{
				Version:     v3.VersionValue,
				FromAddress: jsonrpc.Address(rpcWallet.Address().String()),
				ToAddress:   jsonrpc.Address(cmd.Flag("to").Value.String()),
				StepLimit:   jsonrpc.HexInt(intconv.FormatInt(stepLimit)),
				NetworkID:   jsonrpc.HexInt(intconv.FormatInt(nid)),
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
				if b, err = ZipDirectory(args[0], "__pycache__"); err != nil {
					return fmt.Errorf("fail to zip with directory %s err:%+v", args[0], err)
				}
			} else {
				if b, err = readFile(args[0]); err != nil {
					return fmt.Errorf("fail to read %s err:%+v", args[0], err)
				}
			}
			dataM["content"] = "0x" + hex.EncodeToString(b)
			if dataParams, err := getParamsFromFlags(cmd.Flags()); err != nil {
				return err
			} else if dataParams != nil {
				dataM["params"] = dataParams
			}
			if len(dataM) > 0 {
				param.Data = dataM
			}
			txHash, err := rpcClientSendTx(rpcWallet, param)
			if err != nil {
				return err
			}
			vc.Set("txhash", txHash)
			return JsonPrettyPrintln(os.Stdout, txHash)
		},
	}
	rootCmd.AddCommand(deployCmd)
	deployFlags := deployCmd.Flags()
	deployFlags.String("to", "cx0000000000000000000000000000000000000000", "ToAddress")
	deployFlags.String("content_type", "application/zip",
		"Mime-type of the content")
	deployFlags.String("params", "",
		"raw json string or '@<json file>' or '-' for stdin for parameter JSON")
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
		Args:  ArgsWithDefaultErrorFunc(cobra.ExactArgs(1)),
		RunE: func(cmd *cobra.Command, args []string) error {
			height, err := intconv.ParseInt(args[0], 64)
			if err != nil {
				return err
			}
			param := &server.BlockRequest{Height: common.HexInt64{Value: height}}
			if includeLogs, err := cmd.Flags().GetBool("logs"); err != nil {
				return err
			} else if includeLogs {
				param.Logs = common.HexBool{Value: includeLogs}
			}

			fs, err := cmd.Flags().GetStringArray("filter")
			if err != nil {
				return err
			}
			for _, f := range fs {
				ef := &server.EventFilter{}
				var efBytes []byte
				if strings.HasPrefix(strings.TrimSpace(f), "{") {
					efBytes = []byte(f)
				} else {
					var err error
					if efBytes, err = readFile(f); err != nil {
						return err
					}
				}
				if err := json.Unmarshal(efBytes, ef); err != nil {
					return fmt.Errorf("fail to unmarshal from %s, err:%+v", f, err)
				}
				param.EventFilters = append(param.EventFilters, ef)
			}
			OnInterrupt(rpcClient.Cleanup)
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
	monitorBlockFlags := monitorBlockCmd.Flags()
	monitorBlockFlags.StringArray("filter", nil,
		"EventFilter raw json file or json string")
	monitorBlockFlags.Bool("logs", false, "Includes logs")

	monitorEventCmd := &cobra.Command{
		Use:   "event HEIGHT",
		Short: "MonitorEvent",
		Args:  ArgsWithDefaultErrorFunc(cobra.MaximumNArgs(1)),
		RunE: func(cmd *cobra.Command, args []string) error {
			param := &server.EventRequest{}
			if rawJson := cmd.Flag("raw").Value.String(); rawJson != "" {
				var dataBytes []byte
				if strings.HasPrefix(strings.TrimSpace(rawJson), "{") {
					dataBytes = []byte(rawJson)
				} else {
					var err error
					if dataBytes, err = readFile(rawJson); err != nil {
						return err
					}
				}
				if err := json.Unmarshal(dataBytes, param); err != nil {
					return err
				}
			} else {
				if err := cobra.ExactArgs(1)(cmd, args); err != nil {
					return err
				}
				if err := ValidateFlags(cmd.Flags(), "event"); err != nil {
					return err
				}
			}
			if len(args) > 0 {
				height, err := intconv.ParseInt(args[0], 64)
				if err != nil {
					return err
				}
				param.Height = common.HexInt64{Value: height}
			}
			if includeLogs, err := cmd.Flags().GetBool("logs"); err != nil {
				return err
			} else if includeLogs {
				param.Logs = common.HexBool{Value: includeLogs}
			}

			if sig := cmd.Flag("event").Value.String(); sig != "" {
				param.Signature = sig
			}
			if addr := cmd.Flag("addr").Value.String(); addr != "" {
				param.Addr = common.MustNewAddressFromString(addr)
			}
			if evtIndexed, err := cmd.Flags().GetStringSlice("indexed"); err == nil && len(evtIndexed) > 0 {
				param.Indexed = make([]*string, len(evtIndexed))
				for i, v := range evtIndexed {
					param.Indexed[i] = &v
				}
			}
			if evtData, err := cmd.Flags().GetStringSlice("data"); err == nil && len(evtData) > 0 {
				param.Data = make([]*string, len(evtData))
				for i, v := range evtData {
					param.Data[i] = &v
				}
			}
			OnInterrupt(rpcClient.Cleanup)
			err := rpcClient.MonitorEvent(param, func(v *server.EventNotification) {
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
	monitorEventFlags.String("addr", "", "SCORE Address")
	monitorEventFlags.String("event", "", "Signature of Event")
	monitorEventFlags.StringSlice("indexed", nil, "Indexed Arguments of Event, comma-separated string")
	monitorEventFlags.StringSlice("data", nil, "Not indexed Arguments of Event, comma-separated string")
	monitorEventFlags.String("raw", "", "EventFilter raw json file or json-string")
	monitorEventFlags.Bool("logs", false, "Includes logs")

	monitorBTPCmd := &cobra.Command{
		Use:   "btp HEIGHT",
		Short: "MonitorBTP",
		Args:  ArgsWithDefaultErrorFunc(cobra.ExactArgs(1)),
		RunE: func(cmd *cobra.Command, args []string) error {
			param := &server.BTPRequest{}
			if len(args) > 0 {
				height, err := intconv.ParseInt(args[0], 64)
				if err != nil {
					return err
				}
				param.Height = common.HexInt64{Value: height}
			}

			if nid := cmd.Flag("networkId").Value.String(); nid != "" {
				networkId, err := intconv.ParseInt(nid, 64)
				if err != nil {
					return err
				}
				param.NetworkId = common.HexInt64{Value: networkId}
			}
			if includeProof, err := cmd.Flags().GetBool("proof_flag"); err != nil {
				return err
			} else {
				param.ProofFlag = common.HexBool{Value: includeProof}
			}

			OnInterrupt(rpcClient.Cleanup)
			err := rpcClient.MonitorBtp(param, func(v *server.BTPNotification) {
				JsonPrettyPrintln(os.Stdout, v)
			}, nil)
			if err != nil {
				return err
			}
			return nil
		},
	}
	rootCmd.AddCommand(monitorBTPCmd)
	monitorBTPFlags := monitorBTPCmd.Flags()
	monitorBTPFlags.String("networkId", "",
		"BTP Network ID")
	monitorBTPFlags.Bool("proof_flag", false, "Includes proof")

	return rootCmd
}
