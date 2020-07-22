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

package example;

import score.Address;
import score.ObjectReader;
import score.ObjectWriter;

import java.math.BigInteger;
import java.util.Map;

public class Transaction {
    private final Address destination;
    private final String method;
    private final String params;
    private final BigInteger value;
    private final String description;
    private boolean executed;

    public Transaction(Address destination, String method, String params, BigInteger value, String description) {
        if (destination == null) {
            throw new IllegalArgumentException();
        }
        this.destination = destination;
        this.method = method;
        this.params = params;
        this.value = value;
        this.description = description;
    }

    public static void writeObject(ObjectWriter w, Transaction t) {
        w.beginList(6);
        w.write(t.destination);
        w.writeNullable(
                t.method,
                t.params,
                t.value,
                t.description
        );
        w.write(t.executed);
        w.end();
    }

    public static Transaction readObject(ObjectReader r) {
        r.beginList();
        Transaction t = new Transaction(
                r.readAddress(),
                r.readNullable(String.class),
                r.readNullable(String.class),
                r.readNullable(BigInteger.class),
                r.readNullable(String.class));
        t.setExecuted(r.readBoolean());
        r.end();
        return t;
    }

    public boolean executed() {
        return this.executed;
    }

    public void setExecuted(boolean status) {
        this.executed = status;
    }

    public BigInteger value() {
        return this.value;
    }

    public Address destination() {
        return this.destination;
    }

    public String method() {
        return this.method;
    }

    public String params() {
        return this.params;
    }

    public String description() {
        return this.description;
    }

    public Object[] getConvertedParams() {
        if (params == null || params.equals("")) {
            return null;
        }
        String entries = params.substring(1, params.length() - 1); // strip '[' and ']'
        StringTokenizer entryToken = new StringTokenizer(entries, "{}");
        if (!entryToken.hasMoreTokens()) {
            return new Object[0];
        }
        Object[] ret = new Object[1];
        for (int i = 0; true; i++) {
            String entry = entryToken.nextToken();
            while (",".equals(entry) || " ".equals(entry)) {
                entry = entryToken.nextToken();
            }
            StringTokenizer st = new StringTokenizer(entry, "\":, \t\n");
            String type = null;
            String value = null;
            while (st.hasMoreTokens()) {
                String k = st.nextToken();
                String v = st.nextToken();
                switch (k) {
                    case "type":
                        type = v;
                        break;
                    case "value":
                        value = v;
                        break;
                    case "name":
                        // simply ignore
                        break;
                    default:
                        throw new IllegalArgumentException();
                }
            }
            if (type != null && value != null) {
                ret[i] = convertParam(type, value);
            } else {
                throw new IllegalArgumentException();
            }
            if (entryToken.hasMoreTokens()) {
                // increase the object array
                Object[] dst = new Object[ret.length + 1];
                System.arraycopy(ret, 0, dst, 0, ret.length);
                ret = dst;
            } else {
                break;
            }
        }
        return ret;
    }

    private Object convertParam(String type, String value) {
        switch (type) {
            case "Address":
                return Address.fromString(value);
            case "str":
                return value;
            case "int":
                if (value.startsWith("0x")) {
                    return new BigInteger(value.substring(2), 16);
                }
                return new BigInteger(value);
            case "bool":
                if (value.equals("0x0") || value.equals("false")) {
                    return Boolean.FALSE;
                } else if (value.equals("0x1") || value.equals("true")) {
                    return Boolean.TRUE;
                }
                break;
            case "bytes":
                if (value.startsWith("0x") && (value.length() % 2 == 0)) {
                    String hex = value.substring(2);
                    int len = hex.length() / 2;
                    byte[] bytes = new byte[len];
                    for (int i = 0; i < len; i++) {
                        int j = i * 2;
                        bytes[i] = (byte) Integer.parseInt(hex.substring(j, j + 2), 16);
                    }
                    return bytes;
                }
        }
        throw new IllegalArgumentException("Unknown type");
    }

    @Override
    public String toString() {
        return "Transaction{" +
                "destination=" + destination +
                ", method='" + method + '\'' +
                ", params='" + params + '\'' +
                ", value=" + value +
                ", description='" + description + '\'' +
                ", executed=" + executed +
                '}';
    }

    public Map<String, String> toMap(BigInteger transactionId) {
        return Map.of(
                "_destination", destination.toString(),
                "_method", getSafeString(method),
                "_params", getSafeString(params),
                "_value", (value == null) ? "0x0" : "0x" + value.toString(16),
                "_description", getSafeString(description),
                "_executed", (executed) ? "0x1" : "0x0",
                "_transactionId", "0x" + transactionId.toString(16)
        );
    }

    private String getSafeString(String s) {
        if (s == null) return "";
        return s;
    }
}
