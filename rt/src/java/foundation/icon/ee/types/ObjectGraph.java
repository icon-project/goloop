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

public class ObjectGraph {
    private int nextHash;
    private byte[] graphHash;
    private byte[] graphData;

    public ObjectGraph(int nextHash, byte[] graphData) {
        this(nextHash, null, graphData);
    }

    public ObjectGraph(int nextHash, byte[] graphHash, byte[] graphData) {
        this.nextHash = nextHash;
        this.graphHash = graphHash;
        this.graphData = graphData;
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

    @Override
    public String toString() {
        return "nextHash=" + nextHash +
                ", graphHash=" + (graphHash == null ? null : graphHash.length) +
                ", graphData=" + (graphData == null ? null : graphData.length);
    }
}
