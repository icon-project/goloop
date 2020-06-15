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
import org.aion.avm.core.IExternalState;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;

import java.io.IOException;
import java.math.BigInteger;
import java.util.Map;
import java.util.function.IntConsumer;

public class ExternalState implements IExternalState {
    private static final Logger logger = LoggerFactory.getLogger(ExternalState.class);

    private final EEProxy proxy;
    private final int option;
    private final long blockHeight;
    private final long blockTimestamp;
    private final Address owner;
    private final String code;
    private final FileReader fileReader;
    private ObjectGraph graphCache;
    private final StepCost stepCost;

    ExternalState(EEProxy proxy, int option, String code, FileReader fileReader,
                  BigInteger blockHeight, BigInteger blockTimestamp,
                  Address owner, Map<String, BigInteger> stepCosts) {
        this.proxy = proxy;
        this.option = option;
        this.code = code;
        this.fileReader = fileReader;
        this.blockHeight = blockHeight.longValue();
        this.blockTimestamp = blockTimestamp.longValue();
        this.owner = owner; // owner cannot be null
        this.stepCost = new StepCost(stepCosts);
    }

    @Override
    public byte[] getCode() throws IOException {
        logger.trace("[getCode]");
        return fileReader.readFile(code);
    }

    @Override
    public byte[] getTransformedCode() throws IOException {
        logger.trace("[getTransformedCode]");
        return fileReader.readFile(code);
    }

    @Override
    public void setTransformedCode(byte[] code) {
        logger.trace("[setTransformedCode] len={}", code.length);
        try {
            proxy.setCode(code);
        } catch (IOException e) {
            logger.debug("[setTransformedCode] {}", e.getMessage());
        }
    }

    @Override
    public byte[] getObjectGraph() {
        try {
            ObjectGraph objGraph = null;
            boolean requestFull = true;
            if (graphCache != null) {
                objGraph = proxy.getObjGraph(false);
                if (graphCache.equalGraphData(objGraph)) {
                    objGraph = graphCache;
                    requestFull = false;
                }
            }
            if (requestFull) {
                objGraph = proxy.getObjGraph(true);
                graphCache = objGraph;
            }
            logger.trace("[getObjectGraph] len={}", objGraph.getGraphData().length);
            return objGraph.getRawData();
        } catch (IOException e) {
            logger.debug("[getObjectGraph] {}", e.getMessage());
            return null;
        }
    }

    @Override
    public void putObjectGraph(byte[] data) {
        logger.trace("[putObjectGraph] len={}", data.length);
        try {
            boolean includeGraph = true;
            ObjectGraph objGraph = ObjectGraph.getInstance(data);
            if (graphCache != null && graphCache.equalGraphData(objGraph)) {
                includeGraph = false;
            }
            proxy.setObjGraph(includeGraph, objGraph);
            graphCache = objGraph;
        } catch (IOException e) {
            logger.debug("[putObjectGraph] {}", e.getMessage());
        }
    }

    @Override
    public void putStorage(byte[] key, byte[] value, IntConsumer prevSizeCB) {
        logger.trace("[putStorage] key={} value={}", Bytes.toHexString(key), Bytes.toHexString(value));
        try {
            proxy.setValue(key, value, prevSizeCB);
        } catch (IOException e) {
            logger.debug("[putStorage] {}", e.getMessage());
        }
    }

    @Override
    public boolean waitForCallback() {
        try {
            return proxy.waitForCallback();
        } catch (IOException e) {
            logger.debug("[waitForCallback] {}", e.getMessage());
            return true;
        }
    }

    @Override
    public void waitForCallbacks() {
        try {
            proxy.waitForCallbacks();
        } catch (IOException e) {
            logger.debug("[waitForCallback] {}", e.getMessage());
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
            return null;
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
            return BigInteger.ZERO;
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
    public void log(byte[][] indexed, byte[][] data) {
        try {
            proxy.log(indexed, data);
            logger.trace("[logEvent] {} {}", indexed, data);
        } catch (IOException e) {
            logger.debug("[logEvent] {}", e.getMessage());
        }
    }

    public Result call(Address address, String method, Object[] params, BigInteger value,
                       int stepLimit) {
        try {
            logger.trace("[call] target={} method={} params={} value={} limit={}",
                    address, method, params, value, stepLimit);
            var res = proxy.call(address, method, params, value, stepLimit);
            // TODO: to be removed
            if (res.getRet() instanceof Address) {
                var addr = (Address) res.getRet();
                res = new Result(res.getStatus(), res.getStepUsed(), new score.Address(addr.toByteArray()));
            }
            logger.trace("[call] result={}", res.toString());
            return res;
        } catch (IOException e) {
            logger.debug("[call] {}", e.getMessage());
        }
        return null;
    }

    @Override
    public int getOption() {
        return option;
    }

    @Override
    public StepCost getStepCost() {
        return stepCost;
    }
}
