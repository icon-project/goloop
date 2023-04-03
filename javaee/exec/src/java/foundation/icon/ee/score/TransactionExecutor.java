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

import foundation.icon.ee.Agent;
import foundation.icon.ee.ipc.Connection;
import foundation.icon.ee.ipc.EEProxy;
import foundation.icon.ee.ipc.InvokeResult;
import foundation.icon.ee.types.Address;
import foundation.icon.ee.types.Bytes;
import foundation.icon.ee.types.Method;
import foundation.icon.ee.types.Result;
import foundation.icon.ee.types.Transaction;
import org.aion.avm.core.AvmConfiguration;
import org.aion.avm.core.CommonAvmFactory;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;

import java.io.IOException;
import java.math.BigInteger;
import java.nio.file.Files;
import java.nio.file.Path;
import java.nio.file.Paths;
import java.nio.file.StandardCopyOption;
import java.util.Map;

public class TransactionExecutor {
    private static final Logger logger = LoggerFactory.getLogger(TransactionExecutor.class);
    private static final String CMD_INSTALL = "<init>";

    private final EEProxy proxy;
    private final String uuid;
    private final AvmExecutor avmExecutor;
    private final FileIO fileIO;

    private TransactionExecutor(Connection conn,
                                String uuid,
                                Loader loader,
                                FileIO fileIO,
                                AvmConfiguration conf) {
        this.proxy = new EEProxy(conn);
        this.uuid = uuid;

        proxy.setOnGetApiListener(this::handleGetApi);
        proxy.setOnInvokeListener(this::handleInvoke);
        avmExecutor = CommonAvmFactory.createAvmExecutor(conf, loader);

        this.fileIO = fileIO;
    }

    // TODO : remove me later
    public static TransactionExecutor newInstance(Connection c,
                                                  String uuid) {
        return newInstance(c, uuid, null, null, null);
    }

    public static TransactionExecutor newInstance(Connection c,
                                                  String uuid,
                                                  Loader loader,
                                                  FileIO r,
                                                  AvmConfiguration conf) {
        if (loader == null) {
            loader = new Loader();
        }
        if (r == null) {
            r = DEFAULT_FILE_IO;
        }
        if (conf == null) {
            conf = new AvmConfiguration();
        }
        return new TransactionExecutor(c, uuid, loader, r, conf);
    }

    public void connectAndRunLoop() throws IOException {
        connectAndRunLoop(null);
    }

    public void connectAndRunLoop(Agent agent) throws IOException {
        Agent.agent.set(agent);
        avmExecutor.start();
        try {
            proxy.connect(uuid);
            proxy.handleMessages();
        } finally {
            avmExecutor.shutdown();
            proxy.close();
        }
    }

    public void disconnect() throws IOException {
        proxy.close();
    }

    private Method[] handleGetApi(String path) throws IOException,
            ValidationException {
        logger.trace(">>> path={}", path);
        byte[] jarBytes = fileIO.readFile(
                Path.of(path, ExternalState.CODE_JAR).toString());
        return Validator.validate(jarBytes);
    }

    private InvokeResult handleInvoke(String code, int option, Address from, Address to,
                                      BigInteger value, BigInteger limit,
                                      String method, Object[] params,
                                      Map<String, Object> info, byte[] contractID,
                                      int eid, int nextHash, byte[] graphHash,
                                      int prevEID) throws IOException {
        if (logger.isTraceEnabled()) {
            printInvokeParams(code, option, from, to, value, limit, method, params);
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
        @SuppressWarnings("unchecked")
        Map<String, BigInteger> stepCosts = (Map<String, BigInteger>) info.get(EEProxy.Info.STEP_COSTS);
        long revision = ((BigInteger) info.get(EEProxy.Info.REVISION)).longValue();

        ExternalState kernel = new ExternalState(proxy, option, code,
                fileIO, contractID, blockHeight, blockTimestamp, owner,
                stepCosts, revision, nextHash, graphHash);
        Transaction tx = new Transaction(from, to, value, nonce,
                limit.longValue(), method, params, txHash, txIndex, txTimestamp,
                isInstall);
        Result result = avmExecutor.run(kernel, tx, origin, eid, prevEID);
        return new InvokeResult(result);
    }

    private static final FileIO DEFAULT_FILE_IO = new FileIO() {
        public byte[] readFile(String p) throws IOException {
            return Files.readAllBytes(Paths.get(p));
        }

        public void writeFile(String p, byte[] bytes) throws IOException {
            var temp = Files.createTempFile(Paths.get(p).getParent(), null,
                    null);
            Files.write(temp, bytes);
            Files.move(temp, Paths.get(p), StandardCopyOption.REPLACE_EXISTING);
        }
    };

    private void printInvokeParams(String code, int option, Address from, Address to, BigInteger value,
                                   BigInteger limit, String method, Object[] params) {
        logger.trace(">>> code={}", code);
        logger.trace("    option={}", option);
        logger.trace("    from={}", from);
        logger.trace("      to={}", to);
        logger.trace("    value={}", value);
        logger.trace("    limit={}", limit);
        logger.trace("    method={}", method);
        logger.trace("    params=[");
        for (Object p : params) {
            if (p instanceof byte[]) {
                logger.trace("     - 0x{}", Bytes.toHexString((byte[]) p));
            } else {
                logger.trace("     - {}", p);
            }
        }
        logger.trace("    ]");
    }

    private void printGetInfo(Map<String, Object> info) {
        logger.trace(">>> getInfo: info={}", info);
        logger.trace("    txHash=0x{}", Bytes.toHexString((byte[]) info.get(EEProxy.Info.TX_HASH)));
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
