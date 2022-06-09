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

import foundation.icon.ee.score.FileIO;
import foundation.icon.ee.types.Address;
import foundation.icon.ee.types.Method;
import foundation.icon.ee.util.Containers;

import java.io.IOException;
import java.math.BigInteger;
import java.util.ArrayList;
import java.util.HashMap;
import java.util.Map;

public class Context implements FileIO {
    private final ArrayList<Frame> frames = new ArrayList<>();
    private final OriginFrame originFrame;
    private final Map<String, byte[]> files = new HashMap<>();
    private int eid = 0;
    private int shortCID = 0;
    private boolean commit = false;

    public Context(Address origin) {
        originFrame = new OriginFrame(origin);
        frames.add(originFrame);
    }

    public Context(Address origin, State src) {
        originFrame = new OriginFrame(origin, src);
        frames.add(originFrame);
    }

    public State getState() {
        return originFrame.getState();
    }

    // 0 is first, -1 is last
    private Frame frameAt(int index) {
        if (index >= 0) {
            return frames.get(index);
        } else {
            return frames.get(frames.size() + index);
        }
    }

    private Frame lastFrame() {
        return frames.get(frames.size()-1);
    }

    public Address getOrigin() {
        return originFrame.getAddress();
    }

    public void setOrigin(Address origin) {
        originFrame.setAddress(origin);
    }

    public Address getFrom() {
        return frameAt(-2).getAddress();
    }

    public Address getTo() {
        return lastFrame().getAddress();
    }

    private void beginExternal() {
        eid = 0;
    }

    private void endExternal() {
        lastFrame().getState().gc();
        lastFrame().getState().clearEID();
    }

    public interface SimpleCloseable extends AutoCloseable {
        void close();
    }

    public SimpleCloseable beginExecution() {
        final var size = frames.size();
        if (size == 1) {
            beginExternal();
        } else {
            lastFrame().getContract().setEID(eid);
        }
        ++eid;
        return () -> endExecution(size);
    }

    public void endExecution(int size) {
        if (frames.size() > size) {
            endFrame();
        }
        if (frames.size() == 1) {
            endExternal();
        }
    }

    public SimpleCloseable beginFrame(Address address) {
        return beginFrame(address, null, null, null);
    }

    public SimpleCloseable beginFrame(Address address, String codeID,
            Method[] methods, InvokeHandler ih) {
        var last = lastFrame();
        TxFrame cur;
        if (codeID == null) {
            if (getContract(address) == null) {
                return null;
            }
            cur = TxFrame.newCallFrame(last.getState(), address);
        } else {
            var contractID = Containers.concatArray(address.toByteArray(),
                    BigInteger.valueOf(++shortCID).toByteArray());
            cur = TxFrame.newDeployFrame(last.getState(),
                    address, contractID, codeID, methods, ih);
        }
        frames.add(cur);
        return this::endFrame;
    }

    public void commit(boolean commit) {
        this.commit = commit;
    }

    public void endFrame() {
        lastFrame().getContract().setEID(eid++);
        var frame = frames.remove(frames.size()-1);
        if (commit) {
            lastFrame().setState(frame.getState());
            commit = false;
        }
    }

    public byte[] getStorage(byte[] key) {
        return lastFrame().getAccount().getStorage(key);
    }

    public byte[] setStorage(byte[] key, byte[] value) {
        return lastFrame().getAccount().setStorage(key, value);
    }

    public byte[] removeStorage(byte[] key) {
        return lastFrame().getAccount().removeStorage(key);
    }

    public BigInteger getBalance(Address addr) {
        return lastFrame().getState().getAccount(addr).getBalance();
    }

    public int getContextEID() {
        return eid;
    }

    public Contract getContract(Address address) {
        return lastFrame().getState().getAccount(address).getContract();
    }

    public String getCodeID() {
        return lastFrame().getContract().getCodeID();
    }

    public int getNextHash() {
        return lastFrame().getContract().getNextHash();
    }

    public void setNextHash(int nextHash) {
        lastFrame().getContract().setNextHash(nextHash);
    }

    public byte[] getObjectGraph() {
        return lastFrame().getContract().getObjectGraph();
    }

    public void setObjectGraph(byte[] objectGraph) {
        lastFrame().getContract().setObjectGraph(objectGraph);
    }

    public byte[] getObjectGraphHash() {
        return lastFrame().getContract().getObjectGraphHash();
    }

    public int getEID() {
        return lastFrame().getContract().getEID();
    }

    public int getShortCID() {
        var cid = lastFrame().getContractID();
        return new BigInteger(cid, Address.LENGTH, cid.length-Address.LENGTH)
                .intValue();
    }

    public byte[] getContractID() {
        return lastFrame().getContractID();
    }

    public void writeFile(String path, byte[] data) {
        files.put(path, data.clone());
    }

    public byte[] readFile(String path) throws IOException {
        var data = files.get(path);
        if (data!=null) {
            return data.clone();
        }
        throw new IOException();
    }

    public Method getMethod(Address address, String method) {
        return lastFrame().getState().getAccount(address).getContract()
                .getMethod(method);
    }

    public Method getMethod(String method) {
        return lastFrame().getContract().getMethod(method);
    }
}
