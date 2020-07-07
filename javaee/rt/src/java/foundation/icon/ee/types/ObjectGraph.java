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

import foundation.icon.ee.util.Crypto;

import java.util.Arrays;

public class ObjectGraph {
    private final int nextHash;
    private final byte[] graphHash;
    private final byte[] graphData;

    public ObjectGraph(int nextHash, byte[] graphHash, byte[] graphData) {
        this.nextHash = nextHash;
        this.graphHash = graphHash;
        this.graphData = graphData;
    }

    public ObjectGraph(ObjectGraph base, int newNextHash) {
        this(newNextHash, base.graphHash, base.getGraphData());
    }

    public ObjectGraph(int nextHash, byte[] graphData) {
        this(nextHash, Crypto.sha3_256(graphData), graphData);
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

    public boolean equalGraphData(ObjectGraph objGraph) {
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
}
