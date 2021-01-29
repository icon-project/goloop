/*
 * Copyright 2019 ICON Foundation
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

import foundation.icon.ee.ipc.*;
import foundation.icon.ee.types.Bytes;
import foundation.icon.ee.types.Method;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;

import java.io.IOException;
import java.math.BigInteger;

public class ProxyTest {
    private static final Logger logger = LoggerFactory.getLogger(ProxyTest.class);

    public static void main(String[] args) {
        logger.info("=== ProxyTest ===");
        if (args.length == 2) {
            try {
                Client client = Client.connect(args[0]);
                EEProxy proxy = new EEProxy(client);
                setGetApiHandler(proxy);
                setInvokeHandler(proxy);
                proxy.connect(args[1]);
                proxy.handleMessages();
            } catch (IOException e) {
                e.printStackTrace();
            }
        } else {
            logger.info("Usage: ProxyTest <socket addr> <uuid>");
        }
    }

    private static final String DESC_ADDRESS = "Lscore/Address;";
    private static final String DESC_BIG_INTEGER = "Ljava.math.BigInteger;";

    private static void setGetApiHandler(EEProxy proxy) {
        proxy.setOnGetApiListener(path -> new Method[] {
                Method.newFunction(
                        "balanceOf",
                        Method.Flags.READONLY | Method.Flags.EXTERNAL,
                        new Method.Parameter[] {
                                new Method.Parameter("_owner", DESC_ADDRESS, Method.DataType.ADDRESS)
                        },
                        Method.DataType.INTEGER,
                        "Ljava.math.BigInteger;"
                ),
                Method.newFunction(
                        "name",
                        Method.Flags.READONLY | Method.Flags.EXTERNAL,
                        null,
                        Method.DataType.STRING,
                        "Ljava.lang.String;"
                ),
                Method.newFunction(
                        "transfer",
                        Method.Flags.EXTERNAL,
                        new Method.Parameter[] {
                                new Method.Parameter("_to", DESC_ADDRESS, Method.DataType.ADDRESS),
                                new Method.Parameter("_value", DESC_BIG_INTEGER, Method.DataType.INTEGER),
                                new Method.Parameter("_data", "[B", Method.DataType.BYTES)
                        },
                        Method.DataType.NONE,
                        "V"
                ),
                Method.newFallback(),
                Method.newEvent(
                        "Transfer",
                        3,
                        new Method.Parameter[] {
                                new Method.Parameter("_from", DESC_ADDRESS, Method.DataType.ADDRESS),
                                new Method.Parameter("_to", DESC_ADDRESS, Method.DataType.ADDRESS),
                                new Method.Parameter("_value", DESC_BIG_INTEGER, Method.DataType.INTEGER),
                                new Method.Parameter("_data", "[B", Method.DataType.BYTES)
                        }
                ),
        });
    }

    private static void setInvokeHandler(EEProxy proxy) {
        proxy.setOnInvokeListener((code, isQuery, from, to, value, limit,
                method, params, info, contractID, eid, nextHash, graphHash, prevEID) -> {
            if (logger.isDebugEnabled()) {
                logger.debug(">>> code={}", code);
                logger.debug("    isQuery={}", isQuery);
                logger.debug("    from={}", from);
                logger.debug("      to={}", to);
                logger.debug("    value={}", value);
                logger.debug("    limit={}", limit);
                logger.debug("    method={}", method);

                logger.debug("    params=[");
                for (Object p : params) {
                    logger.debug("     - {}", p);
                }
                logger.debug("    ]");
            }

            if (logger.isDebugEnabled()) {
                logger.debug(">>> getInfo: info={}", info);
                logger.debug("  txHash={}", Bytes.toHexString((byte[]) info.get(EEProxy.Info.TX_HASH)));
                logger.debug("  txIndex={}", info.get(EEProxy.Info.TX_INDEX));
                logger.debug("  txFrom={}", info.get(EEProxy.Info.TX_FROM));
                logger.debug("  txTimestamp={}", info.get(EEProxy.Info.TX_TIMESTAMP));
                logger.debug("  txNonce={}", info.get(EEProxy.Info.TX_NONCE));
                logger.debug("  blockHeight={}", info.get(EEProxy.Info.BLOCK_HEIGHT));
                logger.debug("  blockTimestamp={}", info.get(EEProxy.Info.BLOCK_TIMESTAMP));
                logger.debug("  contractOwner={}", info.get(EEProxy.Info.CONTRACT_OWNER));
                logger.debug("  stepCosts={}", info.get(EEProxy.Info.STEP_COSTS));
            }
            return new InvokeResult(0, BigInteger.ZERO,
                    TypedObj.encodeAny(info.get(EEProxy.Info.STEP_COSTS)));
        });
    }
}
