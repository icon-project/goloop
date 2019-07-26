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
import foundation.icon.tools.ipc.InvokeResult;
import foundation.icon.tools.ipc.Client;
import foundation.icon.tools.ipc.Method;
import foundation.icon.tools.ipc.Proxy;

import java.io.IOException;
import java.math.BigInteger;

public class ProxyTest {

    public static void main(String[] args) {
        System.out.println("=== ProxyTest ===");

        if (args.length == 2) {
            try {
                Client client = Client.connect(args[0]);
                Proxy proxy = new Proxy(client);
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
                proxy.setOnInvokeListener(new Proxy.OnInvokeListener() {
                    @Override
                    public InvokeResult onInvoke(String code, boolean isQuery, Address from, Address to,
                                                 BigInteger value, BigInteger limit, String method,
                                                 Proxy.TypedObj[] params) throws IOException {
                        return new InvokeResult(0, BigInteger.ZERO, Proxy.TypedObj.encodeAny("Test"));
                    }
                });
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
}
