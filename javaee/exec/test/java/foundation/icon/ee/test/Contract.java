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

package foundation.icon.ee.test;

import foundation.icon.ee.types.Address;
import foundation.icon.ee.types.Method;
import foundation.icon.ee.types.Result;
import foundation.icon.ee.util.Crypto;

import java.io.IOException;
import java.math.BigInteger;
import java.util.Map;

public class Contract {
    private final byte[] id;
    private final String codeID;
    private final Method[] methods;
    private int nextHash = 0;
    private byte[] objectGraph = null;
    private byte[] objectGraphHash = null;
    // cleared at the end of external transaction
    private int eid = 0;
    private final InvokeHandler invokeHandler;

    public Contract(byte[] id, String codeID, Method[] methods,
            InvokeHandler ih) {
        this.id = id;
        this.codeID = codeID;
        this.methods = methods;
        if (ih == null) {
            ih = InvokeHandler.defaultHandler();
        }
        this.invokeHandler = ih;
    }

    public Contract(Contract other) {
        this.id = other.id;
        this.codeID = other.codeID;
        this.methods = other.methods;
        this.nextHash = other.nextHash;
        this.objectGraph = other.objectGraph;
        this.objectGraphHash = other.objectGraphHash;
        this.eid = other.eid;
        this.invokeHandler = other.invokeHandler;
    }

    public byte[] getID() {
        return id;
    }

    public String getCodeID() {
        return codeID;
    }

    public Method[] getMethods() {
        return methods;
    }

    public Method getMethod(String name) {
        for (var m : methods) {
            if (m.getName().equals(name)) {
                return m;
            }
        }
        return null;
    }

    public int getNextHash() {
        return nextHash;
    }

    public void setNextHash(int nextHash) {
        this.nextHash = nextHash;
    }

    public byte[] getObjectGraph() {
        return objectGraph;
    }

    public byte[] getObjectGraphHash() {
        return objectGraphHash;
    }

    public void setObjectGraph(byte[] objectGraph) {
        this.objectGraph = objectGraph;
        this.objectGraphHash = Crypto.sha3_256(objectGraph);
    }

    public int getEID() {
        return eid;
    }

    public void setEID(int eid) {
        this.eid = eid;
    }

    public Result invoke(
            ServiceManager sm,
            String code, int flag,
            Address from, Address to, BigInteger value,
            BigInteger stepLimit, String method, Object[] params,
            Map<String, Object> info, byte[] cid, int eid,
            Object[] codeState) throws IOException {
        return invokeHandler.invoke(sm, code, flag,
                from, to, value, stepLimit, method, params, info,
                cid, eid, codeState);
    }
}
