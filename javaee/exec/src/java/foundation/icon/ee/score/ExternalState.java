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
import org.aion.avm.core.IExternalState;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;

import java.io.IOException;
import java.math.BigInteger;

public class ExternalState implements IExternalState {
    private static final Logger logger = LoggerFactory.getLogger(ExternalState.class);

    private final EEProxy proxy;
    private final long blockHeight;
    private final long blockTimestamp;
    private final Address owner;
    private byte[] codeCache;
    private ObjectGraph graphCache;

    ExternalState(EEProxy proxy, byte[] codeBytes, BigInteger blockHeight, BigInteger blockTimestamp, Address owner) {
        this.proxy = proxy;
        this.codeCache = codeBytes;
        this.blockHeight = blockHeight.longValue();
        this.blockTimestamp = blockTimestamp.longValue();
        this.owner = owner; // owner cannot be null
    }

    @Override
    public void commit() {
        logger.trace("[commit]");
    }

    @Override
    public void commitTo(IExternalState externalState) {
        logger.trace("[commitTo] {}", externalState);
        throw new RuntimeException("not implemented");
    }

    @Override
    public IExternalState newChildExternalState() {
        logger.trace("[newChildExternalState]");
        throw new RuntimeException("not implemented");
    }

    @Override
    public void createAccount(Address address) {
        logger.trace("[createAccount] {}", address);
        throw new RuntimeException("not implemented");
    }

    @Override
    public boolean hasAccountState(Address address) {
        logger.trace("[hasAccountState] {}", address);
        throw new RuntimeException("not implemented");
    }

    @Override
    public byte[] getCode(Address address) {
        logger.trace("[getCode] {}", address);
        if (codeCache == null) {
            throw new RuntimeException("code not found");
        }
        return codeCache;
    }

    @Override
    public void putCode(Address address, byte[] code) {
        throw new RuntimeException("should not be called");
    }

    @Override
    public byte[] getTransformedCode(Address address) {
        logger.trace("[getTransformedCode] {}", address);
        if (codeCache == null) {
            throw new RuntimeException("transformed code not found");
        }
        return codeCache;
    }

    @Override
    public void setTransformedCode(Address address, byte[] code) {
        logger.trace("[setTransformedCode] {} len={}", address, code.length);
        try {
            proxy.setCode(code);
            codeCache = code;
        } catch (IOException e) {
            logger.debug("[setTransformedCode] {}", e.getMessage());
        }
    }

    @Override
    public byte[] getObjectGraph(Address address) {
        try {
            ObjectGraph objGraph = null;
            boolean requestFull = true;
            if (graphCache != null) {
                objGraph = proxy.getObjGraph(false);
                if (graphCache.compareTo(objGraph)) {
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
    public void putObjectGraph(Address address, byte[] data) {
        logger.trace("[putObjectGraph] len={}", data.length);
        try {
            boolean includeGraph = true;
            ObjectGraph objGraph = ObjectGraph.getInstance(data);
            if (graphCache != null && graphCache.compareTo(objGraph)) {
                includeGraph = false;
            }
            proxy.setObjGraph(includeGraph, objGraph);
            graphCache = objGraph;
        } catch (IOException e) {
            logger.debug("[putObjectGraph] {}", e.getMessage());
        }
    }

    @Override
    public void putStorage(Address address, byte[] key, byte[] value) {
        logger.trace("[putStorage] key={} value={}", Bytes.toHexString(key), Bytes.toHexString(value));
        try {
            proxy.setValue(key, value);
        } catch (IOException e) {
            logger.debug("[putStorage] {}", e.getMessage());
        }
    }

    @Override
    public void removeStorage(Address address, byte[] key) {
        logger.trace("[removeStorage] key={}", Bytes.toHexString(key));
        try {
            proxy.setValue(key, null);
        } catch (IOException e) {
            logger.debug("[removeStorage] {}", e.getMessage());
        }
    }

    @Override
    public byte[] getStorage(Address address, byte[] key) {
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
    public void deleteAccount(Address address) {
        logger.trace("[deleteStorage] {}", address);
        throw new RuntimeException("not implemented");
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
    public void adjustBalance(Address address, BigInteger amount) {
        logger.trace("[adjustBalance] {} amount={}", address, amount);
        // just ignore this
    }

    @Override
    public BigInteger getNonce(Address address) {
        logger.trace("[getNonce] {}", address);
        throw new RuntimeException("not implemented");
    }

    @Override
    public void incrementNonce(Address address) {
        logger.trace("[incrementNonce] {}", address);
        // just ignore this
    }

    @Override
    public void refundAccount(Address address, BigInteger refund) {
        logger.trace("[refundAccount] {} refund={}", address, refund);
        throw new RuntimeException("not implemented");
    }

    @Override
    public byte[] getBlockHashByHeight(long blockHeight) {
        logger.trace("[getBlockHashByHeight] blockHeight={}", blockHeight);
        throw new RuntimeException("not implemented");
    }

    @Override
    public boolean accountNonceEquals(Address address, BigInteger nonce) {
        logger.trace("[accountNonceEquals] {} nonce={}", address, nonce);
        return true;
    }

    @Override
    public boolean accountBalanceIsAtLeast(Address address, BigInteger amount) {
        logger.trace("[accountBalanceIsAtLeast] {} amount={}", address, amount);
        return true;
    }

    @Override
    public boolean isValidEnergyLimitForCreate(long limit) {
        logger.trace("[isValidEnergyLimitForCreate] limit={}", limit);
        return true;
    }

    @Override
    public boolean isValidEnergyLimitForNonCreate(long limit) {
        logger.trace("[isValidEnergyLimitForNonCreate] limit={}", limit);
        return true;
    }

    @Override
    public boolean destinationAddressIsSafeForThisVM(Address address) {
        logger.trace("[destinationAddressIsSafeForThisVM] {}", address);
        throw new RuntimeException("not implemented");
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
            var res = proxy.call(address, method, params, value, stepLimit);
            // TODO: to be removed
            if (res.getRet() instanceof Address) {
                var addr = (Address) res.getRet();
                res = new Result(res.getStatus(), res.getStepUsed(), new score.Address(addr.toByteArray()));
            }
            logger.trace("[call] status={}", res.getStatus());
            return res;
        } catch (IOException e) {
            logger.debug("[call] {}", e.getMessage());
        }
        return null;
    }
}
