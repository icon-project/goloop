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

package foundation.icon.ee.util;

import org.junit.Test;
import org.junit.jupiter.api.Assertions;

public class LinkedHashMultimapTest {
    public static class TestLHM<K, V> extends LinkedHashMultimap<K, V> {
        private final int cap;

        public TestLHM(int cap) {
            this.cap = cap;
        }

        protected boolean removeEldestEntry(K k, V v) {
            return size() > cap;
        }
    }

    @Test
    public void put() {
        var lhm = new TestLHM<String, String>(3);
        String v1 = "v1";
        Assertions.assertEquals(0, lhm.size());
        Assertions.assertFalse(lhm.contains("k1", v1));
        lhm.put("k1", v1);
        Assertions.assertEquals(1, lhm.size());
        Assertions.assertTrue(lhm.contains("k1", v1));
    }

    @Test
    public void put2() {
        var lhm = new TestLHM<String, Integer>(3);
        Assertions.assertFalse(lhm.contains("k1", 1));
        Assertions.assertEquals(0, lhm.size());
        lhm.put("k1", 1);
        lhm.put("k1", 2);
        Assertions.assertEquals(2, lhm.size());
        Assertions.assertTrue(lhm.contains("k1", 1));
        Assertions.assertTrue(lhm.contains("k1", 2));
    }

    @Test
    public void equality() {
        var lhm = new TestLHM<String, String>(3);
        String v1 = "v1";
        Assertions.assertEquals(0, lhm.size());
        Assertions.assertFalse(lhm.contains("k1", v1));
        lhm.put("k1", v1);
        lhm.put("k1", v1);
        Assertions.assertEquals(1, lhm.size());
        Assertions.assertTrue(lhm.contains("k1", v1));
    }

    @Test
    public void equality2() {
        var lhm = new TestLHM<String, String>(3);
        String v1 = new String("value");
        String v2 = new String("value");
        Assertions.assertNotSame(v1, v2);
        Assertions.assertEquals(0, lhm.size());
        Assertions.assertFalse(lhm.contains("k1", v1));
        Assertions.assertFalse(lhm.contains("k1", v2));
        lhm.put("k1", v1);
        Assertions.assertEquals(1, lhm.size());
        Assertions.assertTrue(lhm.contains("k1", v1));
        Assertions.assertTrue(lhm.contains("k1", v2));
        lhm.put("k1", v2);
        Assertions.assertEquals(1, lhm.size());
        Assertions.assertTrue(lhm.contains("k1", v1));
        Assertions.assertTrue(lhm.contains("k1", v2));
    }

    @Test
    public void remove() {
        var lhm = new TestLHM<String, Integer>(3);
        lhm.put("k1", 1);
        Assertions.assertEquals(1, lhm.size());
        Assertions.assertTrue(lhm.contains("k1", 1));
        lhm.remove("k1", 1);
        Assertions.assertEquals(0, lhm.size());
        Assertions.assertFalse(lhm.contains("k1", 1));
    }

    @Test
    public void remove2() {
        var lhm = new TestLHM<String, Integer>(3);
        lhm.put("k1", 1);
        Assertions.assertEquals(1, lhm.size());
        Assertions.assertTrue(lhm.contains("k1", 1));
        lhm.remove("k1", 0);
        Assertions.assertEquals(1, lhm.size());
        Assertions.assertTrue(lhm.contains("k1", 1));
    }

    @Test
    public void selector() {
        var lhm = new TestLHM<String, Integer>(3);
        lhm.put("k1", 1);
        lhm.put("k1", 2);
        lhm.put("k2", 1);
        lhm.remove("k1", set-> {
            Integer any = null;
            for (var v : set) {
                any = v;
                if (v==1) {
                    return v;
                }
            }
            return any;
        });
        Assertions.assertFalse(lhm.contains("k1", 1));
        Assertions.assertTrue(lhm.contains("k1", 2));
        Assertions.assertTrue(lhm.contains("k2", 1));
    }

    @Test
    public void cap() {
        var lhm = new TestLHM<String, Integer>(3);
        lhm.put("k1", 1);
        lhm.put("k1", 2);
        lhm.put("k1", 3);
        Assertions.assertEquals(3, lhm.size());
        lhm.put("k1", 4);
        Assertions.assertEquals(3, lhm.size());
        Assertions.assertFalse(lhm.contains("k1", 1));
        Assertions.assertTrue(lhm.contains("k1", 2));
        Assertions.assertTrue(lhm.contains("k1", 3));
        Assertions.assertTrue(lhm.contains("k1", 4));
    }

    @Test
    public void cap2() {
        var lhm = new TestLHM<String, Integer>(3);
        lhm.put("k1", 1);
        lhm.put("k2", 2);
        lhm.put("k1", 3);
        lhm.remove("k1", 1);
        lhm.put("k1", 1);
        lhm.put("k1", 4);
        Assertions.assertEquals(3, lhm.size());
        Assertions.assertTrue(lhm.contains("k1", 1));
        Assertions.assertFalse(lhm.contains("k2", 2));
        Assertions.assertTrue(lhm.contains("k1", 3));
        Assertions.assertTrue(lhm.contains("k1", 4));
    }

    @Test
    public void cap3() {
        var lhm = new TestLHM<String, String>(3);
        String v1 = new String("v1");
        String v1_2 = new String("v1");
        String v2 = "v2";
        Assertions.assertNotSame(v1, v1_2);

        lhm.put("k1", v1);
        lhm.put("k1", v2);
        lhm.put("k1", v1_2);
        lhm.put("k2", v1);
        lhm.put("k2", v2);
        Assertions.assertEquals(3, lhm.size());
        Assertions.assertTrue(lhm.contains("k1", v1));
        Assertions.assertFalse(lhm.contains("k1", v2));
        Assertions.assertTrue(lhm.contains("k2", v1));
        Assertions.assertTrue(lhm.contains("k2", v2));
    }
}
