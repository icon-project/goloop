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

package foundation.icon.score;

import foundation.icon.common.Address;
import foundation.icon.common.Bytes;
import foundation.icon.tools.ipc.*;
import org.aion.avm.core.*;
import org.aion.avm.embed.StandardCapabilities;
import org.aion.avm.tooling.ABIUtil;
import org.aion.avm.userlib.CodeAndArguments;
import org.aion.types.AionAddress;
import org.aion.types.Transaction;
import org.aion.types.TransactionResult;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;

import java.io.IOException;
import java.math.BigInteger;
import java.nio.file.Files;
import java.nio.file.Path;
import java.nio.file.Paths;
import java.util.Map;

public class TransactionExecutor {
    private static final Logger logger = LoggerFactory.getLogger(TransactionExecutor.class);
    private static final String CMD_DEPLOY = "<install>";

    private final Proxy proxy;
    private final String uuid;

    private TransactionExecutor(String sockAddr, String uuid) throws IOException {
        Client client = Client.connect(sockAddr);
        this.proxy = new Proxy(client);
        this.uuid = uuid;
    }

    public static TransactionExecutor newInstance(String sockAddr, String uuid) throws IOException {
        TransactionExecutor exec = new TransactionExecutor(sockAddr, uuid);
        exec.setGetApiHandler();
        exec.setInvokeHandler();
        return exec;
    }

    public void connectAndRunLoop() throws IOException {
        proxy.connect(uuid);
        proxy.handleMessages();
    }

    private void setGetApiHandler() {
        proxy.setOnGetApiListener(path -> new Method[] {
                Method.newFunction(
                        "balanceOf",
                        Method.Flags.READONLY | Method.Flags.EXTERNAL,
                        new Method.Parameter[] {
                                new Method.Parameter("_owner", Method.DataType.ADDRESS)
                        },
                        Method.DataType.INTEGER
                ),
        });
    }

    private void setInvokeHandler() {
        proxy.setOnInvokeListener((code, isQuery, from, to, value, limit, method, params) -> {
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

            Map info = (Map) proxy.getInfo();
            if (logger.isDebugEnabled()) {
                logger.debug(">>> getInfo: info={}", info);
                logger.debug("    txHash={}", Bytes.toHexString((byte[]) info.get(Proxy.Info.TX_HASH)));
                logger.debug("    txIndex={}", info.get(Proxy.Info.TX_INDEX));
                logger.debug("    txFrom={}", info.get(Proxy.Info.TX_FROM));
                logger.debug("    txTimestamp={}", info.get(Proxy.Info.TX_TIMESTAMP));
                logger.debug("    txNonce={}", info.get(Proxy.Info.TX_NONCE));
                logger.debug("    blockHeight={}", info.get(Proxy.Info.BLOCK_HEIGHT));
                logger.debug("    blockTimestamp={}", info.get(Proxy.Info.BLOCK_TIMESTAMP));
                logger.debug("    contractOwner={}", info.get(Proxy.Info.CONTRACT_OWNER));
                logger.debug("    stepCosts={}", info.get(Proxy.Info.STEP_COSTS));
            }

            boolean isDeploy = CMD_DEPLOY.equals(method);
            BigInteger blockNumber = (BigInteger) info.get(Proxy.Info.BLOCK_HEIGHT);
            BigInteger blockTimestamp = (BigInteger) info.get(Proxy.Info.BLOCK_TIMESTAMP);
            byte[] txHash = (byte[]) info.get(Proxy.Info.TX_HASH);

            ExternalState kernel = new ExternalState(proxy, code, blockNumber, blockTimestamp);
            Transaction[] contexts = new Transaction[] {
                    getTransactionData(isDeploy, code, from, to, value, limit, method, params, txHash)
            };

            AvmConfiguration config = new AvmConfiguration();
            config.threadCount = 1; // we need only one thread per executor
            if (logger.isDebugEnabled()) {
                config.enableVerboseConcurrentExecutor = true;
                config.enableVerboseContractErrors = true;
            }
            AvmImpl avm = CommonAvmFactory.buildAvmInstanceForConfiguration(new StandardCapabilities(), config);
            try {
                FutureResult[] futures = avm.run(kernel, contexts, ExecutionType.ASSUME_MAINCHAIN, blockNumber.longValue() - 1);
                ResultWrapper result = new ResultWrapper(futures[0].getResult());
                logger.debug("<<< [Result] {}", result);
                Object retVal;
                if (isDeploy) {
                    retVal = result.getContractAddress();
                } else {
                    retVal = result.getDecodedReturnData();
                }
                return new InvokeResult((result.isSuccess()) ? Proxy.Status.SUCCESS : Proxy.Status.FAILURE,
                        result.getEnergyUsed(), TypedObj.encodeAny(retVal));
            } catch (Exception e) {
                e.printStackTrace();
                return new InvokeResult(Proxy.Status.FAILURE, BigInteger.ZERO, TypedObj.encodeAny(e.getMessage()));
            } finally {
                avm.shutdown();
            }
        });
    }

    private Transaction getTransactionData(boolean isDeploy, String code, Address from, Address to,
                                           BigInteger value, BigInteger limit,
                                           String method, Object[] params, byte[] txHash) throws IOException {
        if (isDeploy) {
            byte[] args = ABIUtil.encodeDeploymentArguments(params);
            byte[] txData = new CodeAndArguments(readFile(code), args).encodeToBytes();
            return Transaction.contractCreateTransaction(
                    new AionAddress(from),
                    txHash,
                    BigInteger.valueOf(1),
                    value,
                    txData,
                    limit.longValue(),
                    1L);
        } else {
            byte[] txData = ABIUtil.encodeMethodArguments(method, params);
            return Transaction.contractCallTransaction(
                    new AionAddress(from),
                    new AionAddress(to),
                    txHash,
                    BigInteger.valueOf(1),
                    value,
                    txData,
                    limit.longValue(),
                    1L);
        }
    }

    private byte[] readFile(String code) throws IOException {
        Path path = Paths.get(code);
        byte[] jarBytes;
        try {
            jarBytes = Files.readAllBytes(path);
        } catch (IOException e) {
            throw new IOException("JAR read error: " + e.getMessage());
        }
        return jarBytes;
    }

    private static class ResultWrapper {
        private final TransactionResult result;

        ResultWrapper(TransactionResult result) {
            this.result = result;
        }

        boolean isSuccess() {
            return result.transactionStatus.isSuccess();
        }

        BigInteger getEnergyUsed() {
            return BigInteger.valueOf(result.energyUsed);
        }

        Address getContractAddress() {
            if (!result.transactionStatus.isSuccess()) {
                System.out.println("Contract deployment failed with error " + result.transactionStatus.causeOfError);
                return null;
            }
            return new AionAddress(result.copyOfTransactionOutput().orElseThrow()).toAddress();
        }

        Object getDecodedReturnData() {
            if (!result.transactionStatus.isSuccess()) {
                System.out.println("Contract call failed with error " + result.transactionStatus.causeOfError);
                return null;
            }
            return ABIUtil.decodeOneObject(result.copyOfTransactionOutput().orElseThrow());
        }

        public String toString() {
            return result.toString();
        }
    }
}
