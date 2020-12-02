/*
 * Copyright 2020 ICON Foundation
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

package foundation.icon.ee.types;

import java.math.BigInteger;
import java.util.Arrays;
import java.util.Objects;

public final class Transaction {
    private final Address from;
    private final Address to;
    private final byte[] txHash;
    private final int txIndex;
    private final long txTimestamp;
    private final BigInteger value;
    private final BigInteger nonce;
    private final long limit;
    private final boolean isCreate;
    private final String method;
    private final Object[] params;

    public Transaction(Address from, Address to, BigInteger value, BigInteger nonce, long limit,
                       String method, Object[] params,
                       byte[] txHash, int txIndex, long txTimestamp, boolean isCreate) {
        if (to == null) {
            throw new NullPointerException("No destination");
        }
        if (null == from && txHash != null) {
            throw new NullPointerException("No sender");
        }
        if (null == value) {
            throw new NullPointerException("No value");
        }
        if (null == nonce) {
            throw new NullPointerException("No nonce");
        }
        if (value.compareTo(BigInteger.ZERO) < 0) {
            throw new IllegalArgumentException("Negative value");
        }
        if (nonce.compareTo(BigInteger.ZERO) < 0) {
            throw new IllegalArgumentException("Negative nonce");
        }
        if (limit < 0) {
            throw new IllegalArgumentException("Negative step limit");
        }
        if (null == method && !isCreate) {
            throw new NullPointerException("Null method for call transaction");
        }
        if (null == params) {
            throw new NullPointerException("Null params");
        }

        this.from = from;
        this.to = to;
        if (txHash != null) {
            this.txHash = new byte[txHash.length];
            System.arraycopy(txHash, 0, this.txHash, 0, txHash.length);
        } else {
            this.txHash = null;
        }
        this.txIndex = txIndex;
        this.txTimestamp = txTimestamp;
        this.value = value;
        this.nonce = nonce;
        this.limit = limit;
        this.isCreate = isCreate;
        this.method = method;
        this.params = params;
    }

    public Address getSender() {
        return from;
    }

    public Address getDestination() {
        return to;
    }

    public int getTxIndex() {
        return txIndex;
    }

    public long getTxTimestamp() {
        return txTimestamp;
    }

    public BigInteger getValue() {
        return value;
    }

    public BigInteger getNonce() {
        return nonce;
    }

    public long getLimit() {
        return limit;
    }

    public String getMethod() {
        return method;
    }

    public boolean isCreate() {
        return isCreate;
    }

    public byte[] copyOfTransactionHash() {
        if (txHash == null) {
            return null;
        }
        byte[] copy = new byte[txHash.length];
        System.arraycopy(txHash, 0, copy, 0, txHash.length);
        return copy;
    }

    public Object[] getParams() {
        Object[] paramsCopy = new Object[params.length];
        System.arraycopy(params, 0, paramsCopy, 0, params.length);
        return paramsCopy;
    }

    @Override
    public boolean equals(Object o) {
        if (this == o) return true;
        if (o == null || getClass() != o.getClass()) return false;
        Transaction other = (Transaction) o;
        return txIndex == other.txIndex &&
                txTimestamp == other.txTimestamp &&
                isCreate == other.isCreate &&
                Objects.equals(from, other.from) &&
                Objects.equals(to, other.to) &&
                Arrays.equals(txHash, other.txHash) &&
                Objects.equals(value, other.value) &&
                Objects.equals(nonce, other.nonce) &&
                Objects.equals(limit, other.limit) &&
                Objects.equals(method, other.method) &&
                Arrays.equals(params, other.params);
    }

    @Override
    public int hashCode() {
        int result = Objects.hash(from, to, txIndex, txTimestamp, value, nonce, limit, isCreate, method);
        result = 31 * result + Arrays.hashCode(txHash);
        result = 31 * result + Arrays.hashCode(params);
        return result;
    }

    @Override
    public String toString() {
        return "Transaction{" +
                "from=" + from +
                ", to=" + to +
                ", txHash=" + Arrays.toString(txHash) +
                ", txIndex=" + txIndex +
                ", txTimestamp=" + txTimestamp +
                ", value=" + value +
                ", nonce=" + nonce +
                ", limit=" + limit +
                ", isCreate=" + isCreate +
                ", method='" + method + '\'' +
                ", params=" + Arrays.toString(params) +
                '}';
    }
}
