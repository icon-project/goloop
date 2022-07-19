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

import java.util.HashMap;
import java.util.LinkedHashMap;
import java.util.Map;
import java.util.Set;
import java.util.TreeMap;

/**
 * A read-only data class of RpcObject
 */
public class RpcObject implements RpcItem {
    private final Map<String, RpcItem> items;

    private RpcObject(Map<String, RpcItem> items) {
        this.items = items;
    }

    public Set<String> keySet() {
        return items.keySet();
    }

    public RpcItem getItem(String key) {
        return items.get(key);
    }

    @Override
    public String toString() {
        return "RpcObject(" +
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
        if (!(o instanceof  RpcObject)) return false;
        RpcObject obj = (RpcObject) o;
        return items.equals(obj.items);
    }

    /**
     * Builder for RpcObject
     */
    public static class Builder {

        /**
         * Sort policy of the properties
         */
        public enum Sort {
            NONE,
            KEY,
            INSERT
        }

        private final Map<String, RpcItem> items;

        public Builder() {
            this(Sort.NONE);
        }

        public Builder(Sort sort) {
            switch (sort) {
                case KEY:
                    items = new TreeMap<>();
                    break;
                case INSERT:
                    items = new LinkedHashMap<>();
                    break;
                default:
                    items = new HashMap<>();
                    break;
            }
        }

        public Builder put(String key, RpcItem item) {
            if (item != null) {
                items.put(key, item);
            }
            return this;
        }

        public RpcObject build() {
            return new RpcObject(items);
        }

        public boolean isNullOrEmpty(RpcItem item) {
            return item == null || item.isEmpty();
        }
    }
}
