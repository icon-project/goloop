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

package cli

import (
	"net/http"
	"os"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/icon-project/goloop/client"
	"github.com/icon-project/goloop/server/jsonrpc"
	v3 "github.com/icon-project/goloop/server/v3"
)

func DebugPersistentPreRunE(vc *viper.Viper, dbgClient *client.JsonRpcClient) func(cmd *cobra.Command, args []string) error {
	return func(cmd *cobra.Command, args []string) error {
		if err := ValidateFlagsWithViper(vc, cmd.Flags()); err != nil {
			return err
		}
		*dbgClient = *client.NewJsonRpcClient(&http.Client{}, vc.GetString("uri"))
		return nil
	}
}

func AddDebugRequiredFlags(c *cobra.Command) {
	pFlags := c.PersistentFlags()
	pFlags.String("uri", "", "URI of DEBUG API")
	MarkAnnotationCustom(pFlags, "uri")
}

func NewDebugCmd(parentCmd *cobra.Command, parentVc *viper.Viper) (*cobra.Command, *viper.Viper) {
	var debugClient client.JsonRpcClient
	rootCmd, vc := NewCommand(parentCmd, parentVc, "debug", "DEBUG API")
	rootCmd.PersistentPreRunE = DebugPersistentPreRunE(vc, &debugClient)
	AddDebugRequiredFlags(rootCmd)
	BindPFlags(vc, rootCmd.PersistentFlags())

	traceCmd := &cobra.Command{
		Use:   "trace HASH",
		Short: "Get trace of the transaction",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			param := &v3.TransactionHashParam{
				Hash: jsonrpc.HexBytes(args[0]),
			}
			trace, err := debugClient.Do("debug_getTrace", param, nil)
			if err != nil {
				return err
			}
			return JsonPrettyPrintln(os.Stdout, trace.Result)
		},
	}
	rootCmd.AddCommand(traceCmd)

	return rootCmd, vc
}
