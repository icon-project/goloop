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
import foundation.icon.ee.ipc.Connection;
import foundation.icon.ee.ipc.EEProxy;
import foundation.icon.ee.ipc.InvokeResult;
import foundation.icon.ee.ipc.TypedObj;
import foundation.icon.ee.types.Address;
import foundation.icon.ee.types.Bytes;
import foundation.icon.ee.types.Method;
import foundation.icon.ee.utils.MethodUnpacker;
import org.aion.avm.core.AvmConfiguration;
import org.aion.avm.core.CommonAvmFactory;
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
    private static final AvmConfiguration avmConfig = new AvmConfiguration();

    static {
        if (logger.isDebugEnabled()) {
            avmConfig.enableVerboseContractErrors = true;
        }
    }

    private final EEProxy proxy;
    private final String uuid;
    private final AvmExecutor avmExecutor;
    private final FileReader fileReader;

    private TransactionExecutor(Connection conn,
                                String uuid,
                                Loader loader,
                                FileReader fileReader) throws IOException {
        this.proxy = new EEProxy(conn);
        this.uuid = uuid;

        proxy.setOnGetApiListener(this::handleGetApi);
        proxy.setOnInvokeListener(this::handleInvoke);
        avmExecutor = CommonAvmFactory.createAvmExecutor(avmConfig, loader);

        this.fileReader = fileReader;
    }

    // TODO : remove me later
    public static TransactionExecutor newInstance(Connection c,
                                                  String uuid) throws IOException {
        return newInstance(c, uuid, null, null);
    }

    public static TransactionExecutor newInstance(Connection c,
                                                  String uuid,
                                                  Loader loader,
                                                  FileReader r) throws IOException {
        if (loader == null)
            loader = new Loader();
        if (r == null) {
            r = defaultFileReader;
        }
        return new TransactionExecutor(c, uuid, loader, r);
    }

    public void connectAndRunLoop() throws IOException {
        avmExecutor.start();
        try {
            proxy.connect(uuid);
            proxy.handleMessages();
            proxy.close();
        } finally {
            avmExecutor.shutdown();
        }
    }

    public void disconnect() throws IOException {
        proxy.close();
    }

    private Method[] handleGetApi(String path) throws IOException {
        logger.trace(">>> path={}", path);
        byte[] jarBytes = fileReader.readFile(path);
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
        BigInteger blockHeight = (BigInteger) info.get(EEProxy.Info.BLOCK_HEIGHT);
        BigInteger blockTimestamp = (BigInteger) info.get(EEProxy.Info.BLOCK_TIMESTAMP);
        BigInteger nonce = (BigInteger) info.get(EEProxy.Info.TX_NONCE);
        byte[] txHash = (byte[]) info.get(EEProxy.Info.TX_HASH);
        int txIndex = ((BigInteger) info.get(EEProxy.Info.TX_INDEX)).intValue();
        long txTimestamp = ((BigInteger) info.get(EEProxy.Info.TX_TIMESTAMP)).longValue();
        Address owner = (Address) info.get(EEProxy.Info.CONTRACT_OWNER);
        Address origin = (Address) info.get(EEProxy.Info.TX_FROM);

        byte[] codeBytes = fileReader.readFile(code);
        ExternalState kernel = new ExternalState(proxy, codeBytes, blockHeight, blockTimestamp, owner);
        Transaction tx = getTransactionData(isInstall, from, to, value, nonce, limit, method, params,
                txHash, txIndex, txTimestamp);

        BigInteger energyUsed = BigInteger.ZERO;
        try {
            if (isInstall) {
                // The following is for transformation
                ResultWrapper result = new ResultWrapper(
                        avmExecutor.run(kernel, tx, origin)
                );
                energyUsed = result.getEnergyUsed();
                if (!result.isSuccess()) {
                    throw new RuntimeException(result.getErrorMessage());
                }
                // Prepare another transaction for 'onInstall' itself
                tx = getTransactionData(false, from, to, value, nonce, limit, method, params,
                        txHash, txIndex, txTimestamp);
            }
            // Actual execution of the transaction
            ResultWrapper result = new ResultWrapper(
                    avmExecutor.run(kernel, tx, origin), energyUsed
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
        }
    }

    private Transaction getTransactionData(boolean isInstall, Address from, Address to,
                                           BigInteger value, BigInteger nonce, BigInteger limit,
                                           String method, Object[] params,
                                           byte[] txHash, int txIndex, long txTimestamp) {
        if (to == null) {
            throw new NullPointerException("Cannot create Transaction with null destination!");
        }
        return Transaction.newTransaction(
                from == null ? null : new AionAddress(from),
                new AionAddress(to),
                txHash,
                txIndex,
                txTimestamp,
                value,
                nonce,
                method,
                params,
                limit.longValue(),
                isInstall);
    }

    private static FileReader defaultFileReader = new FileReader() {
        public byte[] readFile(String p) throws IOException {
            Path path = Paths.get(p, CODE_JAR);
            byte[] jarBytes;
            try {
                jarBytes = Files.readAllBytes(path);
            } catch (IOException e) {
                throw new IOException("JAR read error: " + e.getMessage());
            }
            return jarBytes;
        }
    };

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
