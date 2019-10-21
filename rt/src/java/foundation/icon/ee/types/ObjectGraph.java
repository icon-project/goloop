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

package foundation.icon.ee.types;

import java.nio.ByteBuffer;
import java.security.MessageDigest;
import java.security.NoSuchAlgorithmException;
import java.util.Arrays;

public class ObjectGraph {
    private int nextHash;
    private byte[] graphHash;
    private byte[] graphData;

    public ObjectGraph(int nextHash, byte[] graphHash, byte[] graphData) {
        this.nextHash = nextHash;
        this.graphHash = graphHash;
        this.graphData = graphData;
    }

    public static ObjectGraph getInstance(byte[] rawData) {
        ByteBuffer buffer = ByteBuffer.wrap(rawData);
        int nextHash = buffer.getInt();
        byte[] graphData = new byte[buffer.remaining()];
        buffer.get(graphData);
        return new ObjectGraph(nextHash, sha3_256(graphData), graphData);
    }

    public byte[] getRawData() {
        ByteBuffer buffer = ByteBuffer.allocate(4 + this.graphData.length);
        buffer.putInt(this.nextHash);
        buffer.put(this.graphData);
        return buffer.array();
    }

    public int getNextHash() {
        return nextHash;
    }

    public byte[] getGraphHash() {
        return graphHash;
    }

    public byte[] getGraphData() {
        return graphData;
    }

    public boolean compareTo(ObjectGraph objGraph) {
        if (this.graphHash != null && objGraph.graphHash != null) {
            return Arrays.equals(this.graphHash, objGraph.graphHash);
        }
        return Arrays.equals(this.graphData, objGraph.graphData);
    }

    @Override
    public String toString() {
        return "ObjectGraph{" +
                "nextHash=" + nextHash +
                ", graphHash=" + Bytes.toHexString(graphHash) +
                ", graphData=" + (graphData == null ? null : graphData.length) +
                '}';
    }

    // FIXME: Move to other class later
    private static byte[] sha3_256(byte[] msg) {
        try {
            MessageDigest digest = MessageDigest.getInstance("SHA3-256");
            return digest.digest(msg);
        } catch (NoSuchAlgorithmException e) {
            throw new RuntimeException(e);
        }
    }
}
