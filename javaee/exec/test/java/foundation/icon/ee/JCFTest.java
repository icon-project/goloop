package foundation.icon.ee;

import foundation.icon.ee.test.GoldenTest;
import org.junit.jupiter.api.Test;
import score.Context;
import score.annotation.External;

import java.util.List;
import java.util.Map;
import java.util.Set;

public class JCFTest extends GoldenTest {
    public static class Score {
        public static void dumpList(List<?> list) {
            Context.println("list.size=" + list.size());
            for (int i = 0; i < list.size(); i++) {
                Context.println("list.get(" + i + ")=" + list.get(i));
            }
            for (var e : list) {
                Context.println("list element=" + e);
            }
        }

        public static void dumpSet(Set<?> set) {
            Context.println("set.size=" + set.size());
            for (var e : set) {
                Context.println("set element=" + e);
            }
        }

        public static void dumpMap(Map<?, ?> map) {
            Context.println("map.size=" + map.size());
            dumpSet(map.keySet());
            for (var e : map.entrySet()) {
                Context.println("map entry key=" + e.getKey() + " value=" + e.getValue());
                Context.println("map.get(" + e.getKey() + ")=" + map.get(e.getKey()));
            }
        }

        Map<String, Integer> myMap = Map.of();

        @External
        public void setMyMap1() {
            myMap = Map.of("k1", 1, "k2", 2);
        }

        @External
        public void setMyMap2() {
            myMap = Map.of("kkk", 9999);
        }

        @External
        public void dumpMyMap() {
            dumpMap(myMap);
        }

        @External
        public void run() {
            dumpList(List.of());
            dumpList(List.of(0, 1, 2));
            dumpList(List.of(0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11));
            dumpMap(Map.of());
            dumpMap(Map.of("k1", 1, "k2", 2));
            dumpMap(Map.ofEntries(
                    Map.entry("k1", 1),
                    Map.entry("k2", 2)
            ));
            var self = Context.getAddress();
            dumpMap((Map<?, ?>) Context.call(self, "returnMap"));
            dumpMap((Map<?, ?>) Context.call(self, "returnMap2"));
        }

        @External
        public Map<String, Integer> returnMap() {
            return Map.of("k1", 1, "k2", 2);
        }

        @External
        public Map<String, Map<String, List<Integer>>> returnMap2() {
            return Map.of("k1", Map.of("k11", List.of(1, 2, 3)));
        }
    }

    @Test
    public void test() {
        var score = sm.mustDeploy(Score.class);
        score.invoke("run");
        score.invoke("dumpMyMap");
        score.invoke("setMyMap1");
        score.invoke("dumpMyMap");
        score.invoke("setMyMap2");
        score.invoke("dumpMyMap");
    }
}
