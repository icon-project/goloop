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

import java.util.List;

public class DAppRuntimeState {
    private final List<Object> objects;
    private final ObjectGraph graph;

    public DAppRuntimeState(List<Object> objects, ObjectGraph graph) {
        this.objects = objects;
        this.graph = graph;
    }

    public DAppRuntimeState(DAppRuntimeState base, int newNextHash) {
        this(base.objects, new ObjectGraph(base.graph, newNextHash));
    }

    public List<Object> getObjects() {
        return objects;
    }

    public ObjectGraph getGraph() {
        return graph;
    }

    public boolean isAcceptableChangeInReadOnly(DAppRuntimeState newRS) {
        if (objects != null) {
            if (objects.size() != newRS.objects.size()) {
                return false;
            }
            var it = objects.listIterator();
            var it2 = newRS.objects.listIterator();
            while (it.hasNext()) {
                if (it.next() != it2.next()) {
                    return false;
                }
            }
        }
        return graph.equalGraphData(newRS.graph);
    }
}
