/*
 * Copyright 2022 ICON Foundation
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

package pi;

import i.IInstrumentation;
import i.IObject;
import org.junit.jupiter.api.AfterEach;
import org.junit.jupiter.api.BeforeEach;
import org.junit.jupiter.api.Test;
import testutils.TestInstrumentation;

import java.util.Map;

import static org.junit.jupiter.api.Assertions.assertArrayEquals;
import static org.junit.jupiter.api.Assertions.assertTrue;

class UnmodifiableArrayMapTest {
    private final Map<String, String> map = Map.of("Alice", "First", "Bob", "Second", "Charlie", "Third");

    @BeforeEach
    public void setup() throws Exception {
        IInstrumentation.attachedThreadInstrumentation.set(new TestInstrumentation());
    }

    @AfterEach
    public void tearDown() {
        IInstrumentation.attachedThreadInstrumentation.remove();
    }

    private UnmodifiableArrayMap<IObject, IObject> newUnmodifiableMap(Map<String, String> srcMap) {
        var skv = new IObject[srcMap.size() * 2];
        int i = 0;
        for (var e : srcMap.entrySet()) {
            skv[i++] = new s.java.lang.String(e.getKey());
            skv[i++] = new s.java.lang.String(e.getValue());
        }
        return new UnmodifiableArrayMap<>(skv);
    }

    @Test
    void entrySet() {
        var umap = newUnmodifiableMap(map);
        String[] keysArray = new String[map.size()];
        String[] valuesArray = new String[map.size()];
        var entrySet = umap.avm_entrySet();
        var iter = entrySet.avm_iterator();
        int i = 0;
        while (iter.avm_hasNext()) {
            var entry = iter.avm_next();
            keysArray[i] = entry.avm_getKey().toString();
            valuesArray[i] = entry.avm_getValue().toString();
            i++;
        }
        assertArrayEquals(map.keySet().toArray(new String[0]), keysArray);
        assertArrayEquals(map.values().toArray(new String[0]), valuesArray);
    }

    @Test
    void containsKey() {
        var umap = newUnmodifiableMap(map);
        for (var k : map.keySet()) {
            assertTrue(umap.avm_containsKey(new s.java.lang.String(k)));
        }
    }

    @Test
    void containsValue() {
        var umap = newUnmodifiableMap(map);
        for (var v : map.values()) {
            assertTrue(umap.avm_containsValue(new s.java.lang.String(v)));
        }
    }

    @Test
    void equalsTest() {
        var umap = newUnmodifiableMap(map);
        var secondMap = newUnmodifiableMap(map);
        assertTrue(umap.avm_equals(secondMap));
    }
}
