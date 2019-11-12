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

package foundation.icon.ee.score;

import foundation.icon.ee.ipc.Client;
import foundation.icon.ee.ipc.EEProxy;
import foundation.icon.ee.ipc.InvokeResult;
import foundation.icon.ee.ipc.TypedObj;
import foundation.icon.ee.types.Address;
import foundation.icon.ee.types.Bytes;
import foundation.icon.ee.types.Method;
import foundation.icon.ee.utils.MethodUnpacker;
import org.aion.avm.core.AvmConfiguration;
import org.aion.avm.core.CommonAvmFactory;
import org.aion.avm.embed.StandardCapabilities;
import org.aion.avm.utilities.JarBuilder;
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
    private static final String CODE_JAR = "code.jar";
    private static final String CMD_INSTALL = "onInstall";

    private final EEProxy proxy;
    private final String uuid;

    private TransactionExecutor(String sockAddr, String uuid) throws IOException {
        Client client = Client.connect(sockAddr);
        this.proxy = new EEProxy(client);
        this.uuid = uuid;

        proxy.setOnGetApiListener(this::handleGetApi);
        proxy.setOnInvokeListener(this::handleInvoke);
    }

    public static TransactionExecutor newInstance(String sockAddr, String uuid) throws IOException {
        return new TransactionExecutor(sockAddr, uuid);
    }

    public void connectAndRunLoop() throws IOException {
        proxy.connect(uuid);
        proxy.handleMessages();
        proxy.close();
    }

    public void disconnect() throws IOException {
        proxy.close();
    }

    private Method[] handleGetApi(String path) throws IOException {
        logger.trace(">>> path={}", path);
        byte[] jarBytes = readFile(path);
        byte[] apis = JarBuilder.getAPIsBytesFromJAR(jarBytes);
        if (null!=apis) {
            Method[] methods = MethodUnpacker.readFrom(apis);
            for (var m : methods) {
                if (!m.hasValidParams()) {
                    logger.debug("Bad param of API {}!", m.getName());
                    return null;
                }
            }
            return methods;
        }
        logger.debug("No API info found!");
        return null;
    }

    private InvokeResult handleInvoke(String code, boolean isQuery, Address from, Address to,
                                      BigInteger value, BigInteger limit,
                                      String method, Object[] params, Map info) throws IOException {
        if (logger.isTraceEnabled()) {
            printInvokeParams(code, isQuery, from, to, value, limit, method, params);
            printGetInfo(info);
        }
        boolean isInstall = CMD_INSTALL.equals(method);
        BigInteger blockNumber = (BigInteger) info.get(EEProxy.Info.BLOCK_HEIGHT);
        BigInteger blockTimestamp = (BigInteger) info.get(EEProxy.Info.BLOCK_TIMESTAMP);
        byte[] txHash = (byte[]) info.get(EEProxy.Info.TX_HASH);

        byte[] codeBytes = readFile(code);
        ExternalState kernel = new ExternalState(proxy, codeBytes, blockNumber, blockTimestamp);
        Transaction tx = getTransactionData(isInstall, from, to, value, limit, method, params, txHash);

        AvmConfiguration config = new AvmConfiguration();
        if (logger.isDebugEnabled()) {
            config.enableVerboseContractErrors = true;
        }
        AvmExecutor executor = CommonAvmFactory.getAvmInstance(new StandardCapabilities(), config);
        BigInteger energyUsed = BigInteger.ZERO;
        try {
            if (isInstall) {
                // The following is for transformation
                ResultWrapper result = new ResultWrapper(
                        executor.run(kernel, tx, blockNumber.longValue() - 1)
                );
                energyUsed = result.getEnergyUsed();
                if (!result.isSuccess()) {
                    throw new RuntimeException(result.getErrorMessage());
                }
                // Prepare another transaction for 'onInstall' itself
                tx = getTransactionData(false, from, to, value, limit, method, params, txHash);
            }
            // Actual execution of the transaction
            ResultWrapper result = new ResultWrapper(
                    executor.run(kernel, tx, blockNumber.longValue() - 1), energyUsed
            );
            Object retVal = result.getDecodedReturnData();
            return new InvokeResult((result.isSuccess()) ? EEProxy.Status.SUCCESS : EEProxy.Status.FAILURE,
                    result.getEnergyUsed(), TypedObj.encodeAny(retVal));
        } catch (Exception e) {
            String errMsg = e.getMessage();
            if (errMsg == null) {
                errMsg = e.getClass().getName() + " occurred";
            }
            logger.warn("Execution failure", e);
            return new InvokeResult(EEProxy.Status.FAILURE, energyUsed, TypedObj.encodeAny(errMsg));
        } finally {
            executor.shutdown();
        }
    }

    private Transaction getTransactionData(boolean isInstall, Address from, Address to,
                                           BigInteger value, BigInteger limit,
                                           String method, Object[] params, byte[] txHash) {
        if (isInstall) {
            return Transaction.contractCreateTransaction(
                    new AionAddress(from),
                    txHash,
                    BigInteger.valueOf(1),
                    value,
                    method,
                    params,
                    limit.longValue(),
                    1L);
        } else {
            return Transaction.contractCallTransaction(
                    from == null ? null : new AionAddress(from),
                    new AionAddress(to),
                    txHash,
                    BigInteger.valueOf(1),
                    value,
                    method,
                    params,
                    limit.longValue(),
                    1L);
        }
    }

    private byte[] readFile(String code) throws IOException {
        Path path = Paths.get(code, CODE_JAR);
        byte[] jarBytes;
        try {
            jarBytes = Files.readAllBytes(path);
        } catch (IOException e) {
            throw new IOException("JAR read error: " + e.getMessage());
        }
        return jarBytes;
    }

    private Object[] getConvertedParams(Object[] params) {
        Object[] convertedParams = new Object[params.length];
        for (int i = 0; i < params.length; i++) {
            Object obj = params[i];
            if (obj instanceof Address) {
                Address address = (Address) obj;
                convertedParams[i] = new avm.Address(new AionAddress(address).toByteArray());
            } else {
                convertedParams[i] = obj;
            }
        }
        return convertedParams;
    }

    private static class ResultWrapper {
        private final TransactionResult result;
        private final long energyUsed;

        ResultWrapper(TransactionResult result) {
            this(result, BigInteger.ZERO);
        }

        ResultWrapper(TransactionResult result, BigInteger energyUsed) {
            this.result = result;
            this.energyUsed = energyUsed.longValue();
        }

        boolean isSuccess() {
            return result.transactionStatus.isSuccess();
        }

        BigInteger getEnergyUsed() {
            return BigInteger.valueOf(result.energyUsed + this.energyUsed);
        }

        Object getDecodedReturnData() {
            if (!isSuccess()) {
                return null;
            }
            return result.copyOfTransactionOutput();
        }

        String getErrorMessage() {
            return result.transactionStatus.causeOfError;
        }

        public String toString() {
            return result.toString();
        }
    }

    private void printInvokeParams(String code, boolean isQuery, Address from, Address to, BigInteger value,
                                   BigInteger limit, String method, Object[] params) {
        logger.trace(">>> code={}", code);
        logger.trace("    isQuery={}", isQuery);
        logger.trace("    from={}", from);
        logger.trace("      to={}", to);
        logger.trace("    value={}", value);
        logger.trace("    limit={}", limit);
        logger.trace("    method={}", method);
        logger.trace("    params=[");
        for (Object p : params) {
            logger.trace("     - {}", p);
        }
        logger.trace("    ]");
    }

    private void printGetInfo(Map info) {
        logger.trace(">>> getInfo: info={}", info);
        logger.trace("    txHash={}", Bytes.toHexString((byte[]) info.get(EEProxy.Info.TX_HASH)));
        logger.trace("    txIndex={}", info.get(EEProxy.Info.TX_INDEX));
        logger.trace("    txFrom={}", info.get(EEProxy.Info.TX_FROM));
        logger.trace("    txTimestamp={}", info.get(EEProxy.Info.TX_TIMESTAMP));
        logger.trace("    txNonce={}", info.get(EEProxy.Info.TX_NONCE));
        logger.trace("    blockHeight={}", info.get(EEProxy.Info.BLOCK_HEIGHT));
        logger.trace("    blockTimestamp={}", info.get(EEProxy.Info.BLOCK_TIMESTAMP));
        logger.trace("    contractOwner={}", info.get(EEProxy.Info.CONTRACT_OWNER));
        logger.trace("    stepCosts={}", info.get(EEProxy.Info.STEP_COSTS));
    }
}
