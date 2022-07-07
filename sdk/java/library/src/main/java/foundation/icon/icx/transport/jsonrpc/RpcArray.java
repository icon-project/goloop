/*
 * Copyright 2018 ICON Foundation
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

package foundation.icon.icx.transport.jsonrpc;

import java.util.ArrayList;
import java.util.Iterator;
import java.util.List;

/**
 * A read-only data class of RpcArray
 */
public class RpcArray implements RpcItem, Iterable<RpcItem> {
    private final List<RpcItem> items;

    private RpcArray(List<RpcItem> items) {
        this.items = items;
    }

    public Iterator<RpcItem> iterator() {
        return items.iterator();
    }

    public RpcItem get(int index) {
        return items.get(index);
    }

    public int size() {
        return items.size();
    }

    public List<RpcItem> asList() {
        return new ArrayList<>(items);
    }

    @Override
    public String toString() {
        return "RpcArray(" +
                "items=" + items +
                ')';
    }

    @Override
    public boolean isEmpty() {
        return items == null || items.isEmpty();
    }

    @Override
    public boolean equals(Object o) {
        if (this == o) return true;
        if (!(o instanceof  RpcArray)) return false;
        RpcArray obj = (RpcArray) o;
        return items.equals(obj.items);
    }
    /**
     * Builder for RpcArray
     */
    public static class Builder {

        private final List<RpcItem> items;

        public Builder() {
            items = new ArrayList<>();
        }

        public Builder add(RpcItem item) {
            items.add(item);
            return this;
        }

        public RpcArray build() {
            return new RpcArray(items);
        }
    }
}
