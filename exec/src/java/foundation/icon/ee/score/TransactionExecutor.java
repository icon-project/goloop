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
import org.aion.avm.core.*;
import org.aion.avm.embed.StandardCapabilities;
import org.aion.avm.tooling.ABIUtil;
import org.aion.avm.userlib.CodeAndArguments;
import org.aion.types.AionAddress;
import org.aion.types.Transaction;
import org.aion.types.TransactionResult;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;

import java.io.ByteArrayInputStream;
import java.io.IOException;
import java.math.BigInteger;
import java.nio.file.Files;
import java.nio.file.Path;
import java.nio.file.Paths;
import java.util.Map;
import java.util.jar.JarEntry;
import java.util.jar.JarInputStream;

public class TransactionExecutor {
    private static final Logger logger = LoggerFactory.getLogger(TransactionExecutor.class);
    private static final String CODE_JAR = "code.jar";
    private static final String CMD_DEPLOY = "<install>";
    private static final String APIS_NAME = "META-INF/APIS";

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
        logger.debug(">>> path={}", path);
        byte[] jarBytes = readFile(path);
        JarInputStream jis = new JarInputStream(new ByteArrayInputStream(jarBytes), true);
        JarEntry entry;
        while ((entry = jis.getNextJarEntry()) != null) {
            if (entry.getName().equals(APIS_NAME)) {
                byte[] buffer = jis.readAllBytes();
                return MethodUnpacker.readFrom(buffer);
            }
        }
        logger.debug("No API info found!");
        return null;
    }

    private InvokeResult handleInvoke(String code, boolean isQuery, Address from, Address to,
                                      BigInteger value, BigInteger limit,
                                      String method, Object[] params) throws IOException {
        if (logger.isDebugEnabled()) {
            printInvokeParams(code, isQuery, from, to, value, limit, method, params);
        }

        Map info = (Map) proxy.getInfo();
        if (logger.isDebugEnabled()) {
            printGetInfo(info);
        }

        boolean isDeploy = CMD_DEPLOY.equals(method);
        BigInteger blockNumber = (BigInteger) info.get(EEProxy.Info.BLOCK_HEIGHT);
        BigInteger blockTimestamp = (BigInteger) info.get(EEProxy.Info.BLOCK_TIMESTAMP);
        byte[] txHash = (byte[]) info.get(EEProxy.Info.TX_HASH);

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
            Object retVal;
            if (isDeploy) {
                retVal = result.getContractAddress();
            } else {
                retVal = result.getDecodedReturnData();
            }
            return new InvokeResult((result.isSuccess()) ? EEProxy.Status.SUCCESS : EEProxy.Status.FAILURE,
                    result.getEnergyUsed(), TypedObj.encodeAny(retVal));
        } catch (Exception e) {
            e.printStackTrace();
            return new InvokeResult(EEProxy.Status.FAILURE, BigInteger.ZERO, TypedObj.encodeAny(e.getMessage()));
        } finally {
            avm.shutdown();
        }
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
            byte[] txData = ABIUtil.encodeMethodArguments(method, getConvertedParams(params));
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

    private void printInvokeParams(String code, boolean isQuery, Address from, Address to, BigInteger value,
                                   BigInteger limit, String method, Object[] params) {
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

    private void printGetInfo(Map info) {
        logger.debug(">>> getInfo: info={}", info);
        logger.debug("    txHash={}", Bytes.toHexString((byte[]) info.get(EEProxy.Info.TX_HASH)));
        logger.debug("    txIndex={}", info.get(EEProxy.Info.TX_INDEX));
        logger.debug("    txFrom={}", info.get(EEProxy.Info.TX_FROM));
        logger.debug("    txTimestamp={}", info.get(EEProxy.Info.TX_TIMESTAMP));
        logger.debug("    txNonce={}", info.get(EEProxy.Info.TX_NONCE));
        logger.debug("    blockHeight={}", info.get(EEProxy.Info.BLOCK_HEIGHT));
        logger.debug("    blockTimestamp={}", info.get(EEProxy.Info.BLOCK_TIMESTAMP));
        logger.debug("    contractOwner={}", info.get(EEProxy.Info.CONTRACT_OWNER));
        logger.debug("    stepCosts={}", info.get(EEProxy.Info.STEP_COSTS));
    }
}
