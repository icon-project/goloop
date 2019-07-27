/*
 * Copyright (c) 2019 ICON Foundation
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

import foundation.icon.common.Address;
import foundation.icon.tools.ipc.Client;
import foundation.icon.tools.ipc.InvokeResult;
import foundation.icon.tools.ipc.Method;
import foundation.icon.tools.ipc.Proxy;

import java.io.IOException;
import java.math.BigInteger;
import java.util.HashMap;

public class ProxyTest {

    private static final boolean DEBUG = true;

    public static void main(String[] args) {
        System.out.println("=== ProxyTest ===");

        if (args.length == 2) {
            try {
                Client client = Client.connect(args[0]);
                Proxy proxy = new Proxy(client);
                setGetApiHandler(proxy);
                setInvokeHandler(proxy);
                proxy.connect(args[1]);
                proxy.handleMessages();
            } catch (IOException e) {
                e.printStackTrace();
            }
        } else {
            System.out.println("Usage: ProxyTest <socket addr> <uuid>");
        }

        System.out.println("=== END ===");
    }

    private static void setGetApiHandler(Proxy proxy) {
        proxy.setOnGetApiListener(path -> new Method[] {
                Method.newFunction(
                        "balanceOf",
                        Method.Flags.READONLY | Method.Flags.EXTERNAL,
                        new Method.Parameter[] {
                                new Method.Parameter("_owner", Method.DataType.ADDRESS)
                        },
                        Method.DataType.INTEGER
                ),
                Method.newFunction(
                        "name",
                        Method.Flags.READONLY | Method.Flags.EXTERNAL,
                        null,
                        Method.DataType.STRING
                ),
                Method.newFunction(
                        "transfer",
                        Method.Flags.EXTERNAL,
                        new Method.Parameter[] {
                                new Method.Parameter("_to", Method.DataType.ADDRESS),
                                new Method.Parameter("_value", Method.DataType.INTEGER),
                                new Method.Parameter("_data", Method.DataType.BYTES)
                        },
                        Method.DataType.NONE
                ),
                Method.newFallback(),
                Method.newEvent(
                        "Transfer",
                        3,
                        new Method.Parameter[] {
                                new Method.Parameter("_from", Method.DataType.ADDRESS),
                                new Method.Parameter("_to", Method.DataType.ADDRESS),
                                new Method.Parameter("_value", Method.DataType.INTEGER),
                                new Method.Parameter("_data", Method.DataType.BYTES)
                        }
                ),
        });
    }

    private static void setInvokeHandler(Proxy proxy) {
        proxy.setOnInvokeListener((code, isQuery, from, to, value, limit, method, params) -> {
            if (DEBUG) {
                System.out.println(">>> code=" + code);
                System.out.println("    isQuery=" + isQuery);
                System.out.println("    from=" + from);
                System.out.println("      to=" + to);
                System.out.println("    value=" + value);
                System.out.println("    limit=" + limit);
                System.out.println("    method=" + method);

                System.out.println("    params=[");
                for (Object p : params) {
                    System.out.println("     - " + p);
                }
                System.out.println("    ]");
            }

            HashMap info = (HashMap) proxy.getInfo();
            if (DEBUG) {
                System.out.println(">>> getInfo: info=" + info);
                System.out.println("  txHash=" + info.get(Proxy.Info.TX_HASH));
                System.out.println("  txIndex=" + info.get(Proxy.Info.TX_INDEX));
                System.out.println("  txFrom=" + info.get(Proxy.Info.TX_FROM));
                System.out.println("  txTimestamp=" + info.get(Proxy.Info.TX_TIMESTAMP));
                System.out.println("  txNonce=" + info.get(Proxy.Info.TX_NONCE));
                System.out.println("  blockHeight=" + info.get(Proxy.Info.BLOCK_HEIGHT));
                System.out.println("  blockTimestamp=" + info.get(Proxy.Info.BLOCK_TIMESTAMP));
                System.out.println("  contractOwner=" + info.get(Proxy.Info.CONTRACT_OWNER));
                System.out.println("  stepCosts=" + info.get(Proxy.Info.STEP_COSTS));
            }
            return new InvokeResult(0, BigInteger.ZERO, Proxy.TypedObj.encodeAny("Test"));
        });
    }
}
