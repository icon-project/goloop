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

import foundation.icon.ee.ipc.EEProxy;
import foundation.icon.ee.types.Address;
import foundation.icon.ee.types.Bytes;
import foundation.icon.ee.types.ObjectGraph;
import foundation.icon.ee.types.Result;
import foundation.icon.ee.types.StepCost;
import i.RuntimeAssertionError;
import org.aion.avm.core.IExternalState;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;

import java.io.IOException;
import java.math.BigInteger;
import java.nio.file.Path;
import java.util.Map;
import java.util.function.IntConsumer;

public class ExternalState implements IExternalState {
    private static final Logger logger = LoggerFactory.getLogger(ExternalState.class);
    public static final String CODE_JAR = "code.jar";
    public static final String TRANSFORMED_JAR = "transformed.jar";

    private final EEProxy proxy;
    private final int option;
    private final long blockHeight;
    private final long blockTimestamp;
    private final Address owner;
    private final String codePath;
    private final FileIO fileIO;
    private final byte[] contractID;
    private final StepCost stepCost;
    private final long revision;
    private final int nextHash;
    private final byte[] graphHash;
    private int feeProportion;

    ExternalState(EEProxy proxy, int option, String codePath,
                  FileIO fileIO, byte[] contractID, BigInteger blockHeight,
                  BigInteger blockTimestamp, Address owner,
                  Map<String, BigInteger> stepCosts, long revision, int nextHash,
                  byte[] graphHash) {
        this.proxy = proxy;
        this.option = option;
        this.codePath = codePath;
        this.fileIO = fileIO;
        this.contractID = contractID;
        this.blockHeight = blockHeight.longValue();
        this.blockTimestamp = blockTimestamp.longValue();
        this.owner = owner; // owner cannot be null
        this.stepCost = new StepCost(stepCosts);
        this.revision = revision;
        this.nextHash = nextHash;
        this.graphHash = graphHash;
    }

    public String getCodeID() {
        return Path.of(codePath).getFileName().toString();
    }

    @Override
    public byte[] getCode() {
        logger.trace("[getCode]");
        try {
            return fileIO.readFile(Path.of(codePath, CODE_JAR).toString());
        } catch (IOException e) {
            logger.debug("[getCode] {}", e.getMessage());
            throw RuntimeAssertionError.unexpected(e);
        }
    }

    @Override
    public byte[] getTransformedCode() throws IOException {
        logger.trace("[getTransformedCode]");
        return fileIO.readFile(Path.of(codePath, TRANSFORMED_JAR).toString());
    }

    @Override
    public void setTransformedCode(byte[] code) {
        logger.trace("[setTransformedCode] len={}", code.length);
        try {
            fileIO.writeFile(Path.of(codePath, TRANSFORMED_JAR).toString(),
                    code);
        } catch (IOException e) {
            logger.debug("[setTransformedCode] {}", e.getMessage());
            RuntimeAssertionError.unexpected(e);
        }
    }

    public byte[] getContractID() {
        return contractID;
    }

    @Override
    public ObjectGraph getObjectGraph() {
        try {
            var objGraph = proxy.getObjGraph(true);
            logger.trace("[getObjectGraph] len={}", objGraph.getGraphData().length);
            return objGraph;
        } catch (IOException e) {
            logger.debug("[getObjectGraph] {}", e.getMessage());
            throw RuntimeAssertionError.unexpected(e);
        }
    }

    @Override
    public byte[] getObjectGraphHash() {
        return graphHash;
    }

    @Override
    public int getNextHash() {
        return nextHash;
    }

    @Override
    public void putObjectGraph(ObjectGraph objGraph) {
        logger.trace("[putObjectGraph] len={}", objGraph.getGraphData().length);
        try {
            boolean includeGraph = true;
            proxy.setObjGraph(includeGraph, objGraph);
        } catch (IOException e) {
            logger.debug("[putObjectGraph] {}", e.getMessage());
            RuntimeAssertionError.unexpected(e);
        }
    }

    @Override
    public void putStorage(byte[] key, byte[] value, IntConsumer prevSizeCB) {
        logger.trace("[putStorage] key={} value={}", Bytes.toHexString(key), Bytes.toHexString(value));
        try {
            proxy.setValue(key, value, prevSizeCB);
        } catch (IOException e) {
            logger.debug("[putStorage] {}", e.getMessage());
            RuntimeAssertionError.unexpected(e);
        }
    }

    @Override
    public boolean waitForCallback() {
        try {
            return proxy.waitForCallback();
        } catch (IOException e) {
            logger.debug("[waitForCallback] {}", e.getMessage());
            throw RuntimeAssertionError.unexpected(e);
        }
    }

    @Override
    public void waitForCallbacks() {
        try {
            proxy.waitForCallbacks();
        } catch (IOException e) {
            logger.debug("[waitForCallback] {}", e.getMessage());
            RuntimeAssertionError.unexpected(e);
        }
    }

    @Override
    public void limitPendingCallbackLength() {
        try {
            proxy.limitPendingCallbackLength();
        } catch (IOException e) {
            logger.debug("[limitPendingCallbackLength] {}", e.getMessage());
            RuntimeAssertionError.unexpected(e);
        }
    }

    @Override
    public byte[] getStorage(byte[] key) {
        try {
            byte[] value = proxy.getValue(key);
            logger.trace("[getStorage] key={} value={}", Bytes.toHexString(key), Bytes.toHexString(value));
            return value;
        } catch (IOException e) {
            logger.debug("[getStorage] {}", e.getMessage());
            throw RuntimeAssertionError.unexpected(e);
        }
    }

    @Override
    public BigInteger getBalance(Address address) {
        try {
            BigInteger balance = proxy.getBalance(address);
            logger.trace("[getBalance] {} balance={}", address, balance);
            return balance;
        } catch (IOException e) {
            logger.debug("[getBalance] {}", e.getMessage());
            throw RuntimeAssertionError.unexpected(e);
        }
    }

    @Override
    public long getBlockHeight() {
        logger.trace("[getBlockHeight] ret={}", blockHeight);
        return blockHeight;
    }

    @Override
    public long getBlockTimestamp() {
        logger.trace("[getBlockTimestamp] ret={}", blockTimestamp);
        return blockTimestamp;
    }

    @Override
    public Address getOwner() {
        logger.trace("[getOwner] ret={}", owner);
        return owner;
    }

    @Override
    public void event(byte[][] indexed, byte[][] data) {
        try {
            proxy.event(indexed, data);
        } catch (IOException e) {
            logger.debug("[logEvent] {}", e.getMessage());
            RuntimeAssertionError.unexpected(e);
        }
    }

    @Override
    public int getFeeSharingProportion() {
        return this.feeProportion;
    }

    @Override
    public void setFeeSharingProportion(int proportion) {
        this.feeProportion = proportion;
        try {
            proxy.setFeeSharingProportion(proportion);
        } catch (IOException e) {
            logger.debug("[setFeeSharingProportion] {}", e.getMessage());
            RuntimeAssertionError.unexpected(e);
        }
    }

    @Override
    public Result call(Address address, BigInteger value, long stepLimit,
                       String dataType, Object dataObj) {
        try {
            logger.trace("[call] target={} value={} limit={} dataType={} dataObj={} ",
                    address, value, stepLimit, dataType, dataObj);
            var res = proxy.call(address, value, stepLimit, dataType, dataObj);
            logger.trace("[call] result={}", res.toString());
            return res;
        } catch (IOException e) {
            logger.debug("[call] {}", e.getMessage());
            throw RuntimeAssertionError.unexpected(e);
        }
    }

    @Override
    public int getOption() {
        return option;
    }

    @Override
    public StepCost getStepCost() {
        return stepCost;
    }

    @Override
    public long getRevision() {
        return revision;
    }
}
