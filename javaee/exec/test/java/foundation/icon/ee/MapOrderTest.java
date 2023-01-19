/*
 * Copyright 2021 ICON Foundation
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

package foundation.icon.ee;

import foundation.icon.ee.test.NoDebugTest;
import foundation.icon.ee.test.SimpleTest;
import org.junit.jupiter.api.Test;
import score.Address;
import score.Context;
import score.annotation.External;

import java.util.Iterator;
import java.util.Map;

public class MapOrderTest extends SimpleTest {
    public static class SimpleMapMaker {
        @External
        public Map<String, String> make() {
            return Map.of(
                    "k5", "v5",
                    "k2", "v2",
                    "k4", "v4",
                    "k1", "v1",
                    "k3", "v3"
            );
        }
    }

    public static class Struct {
        public String getK5() { return "v5"; }
        public String getK2() { return "v2"; }
        public String getK4() { return "v4"; }
        public String getK1() { return "v1"; }
        public String getK3() { return "v3"; }
    }

    public static class StructMapMaker {
        @External
        public Struct make() {
            return new Struct();
        }
    }

    public static class MapTaker {
        private void requireKeyValue(Map.Entry<String, String> e, String k,
                String v) {
            Context.println("expected k:" + k + " v:" + v);
            Context.println("observed k:" + e.getKey() + " v:" + e.getValue());
            Context.require(e.getKey().equals(k));
            Context.require(e.getValue().equals(v));
        }

        @External
        public void take(Address maker) {
            var map = Context.call(Map.class, maker, "make");
            Iterator<Map.Entry<String, String>> it = map.entrySet().iterator();
            requireKeyValue(it.next(), "k5", "v5");
            requireKeyValue(it.next(), "k2", "v2");
            requireKeyValue(it.next(), "k4", "v4");
            requireKeyValue(it.next(), "k1", "v1");
            requireKeyValue(it.next(), "k3", "v3");
        }

        @External
        public void takeSorted(Address maker) {
            var map = Context.call(Map.class, maker, "make");
            Iterator<Map.Entry<String, String>> it = map.entrySet().iterator();
            requireKeyValue(it.next(), "k1", "v1");
            requireKeyValue(it.next(), "k2", "v2");
            requireKeyValue(it.next(), "k3", "v3");
            requireKeyValue(it.next(), "k4", "v4");
            requireKeyValue(it.next(), "k5", "v5");
        }
    }

    @Test
    void simpleReturnValue() {
        var maker = sm.mustDeploy(SimpleMapMaker.class);
        var taker = sm.mustDeploy(MapTaker.class);
        taker.invoke("take", maker.getAddress());
    }

    public static class StructReturnValueTest extends NoDebugTest {
        @Test
        void structReturnValue() {
            var maker = sm.mustDeploy(new Class<?>[]{
                    StructMapMaker.class, Struct.class
            });
            var taker = sm.mustDeploy(MapTaker.class);
            taker.invoke("takeSorted", maker.getAddress());
        }
    }
}
