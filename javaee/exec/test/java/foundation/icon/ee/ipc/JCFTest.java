package foundation.icon.ee.ipc;

import avm.Blockchain;
import foundation.icon.ee.test.GoldenTest;
import foundation.icon.ee.tooling.abi.External;
import org.junit.jupiter.api.Test;

import java.util.List;
import java.util.Map;
import java.util.Set;

public class JCFTest extends GoldenTest {
    public static class Score {
        public static void dumpList(List<?> list) {
            Blockchain.println("list.size=" + list.size());
            for (int i = 0; i < list.size(); i++) {
                Blockchain.println("list.get(" + i + ")=" + list.get(i));
            }
            for (var e : list) {
                Blockchain.println("list element=" + e);
            }
        }

        public static void dumpSet(Set<?> set) {
            Blockchain.println("set.size=" + set.size());
            for (var e : set) {
                Blockchain.println("set element=" + e);
            }
        }

        public static void dumpMap(Map<?, ?> map) {
            Blockchain.println("map.size=" + map.size());
            dumpSet(map.keySet());
            for (var e : map.entrySet()) {
                Blockchain.println("map entry key=" + e.getKey() + " value=" + e.getValue());
                Blockchain.println("map.get(" + e.getKey() + ")=" + map.get(e.getKey()));
            }
        }

        static Map<String, Integer> myMap = Map.of();

        @External
        public static void setMyMap1() {
            myMap = Map.of("k1", 1, "k2", 2);
        }

        @External
        public static void setMyMap2() {
            myMap = Map.of("kkk", 9999);
        }

        @External
        public static void dumpMyMap() {
            dumpMap(myMap);
        }

        @External
        public static void run() {
            dumpList(List.of());
            dumpList(List.of(0, 1, 2));
            dumpList(List.of(0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11));
            dumpMap(Map.of());
            dumpMap(Map.of("k1", 1, "k2", 2));
            dumpMap(Map.ofEntries(
                    Map.entry("k1", 1),
                    Map.entry("k2", 2)
            ));
        }
    }

    @Test
    public void test() {
        var score = sm.deploy(Score.class);
        score.invoke("run");
        score.invoke("dumpMyMap");
        score.invoke("setMyMap1");
        score.invoke("dumpMyMap");
        score.invoke("setMyMap2");
        score.invoke("dumpMyMap");
        // test object return
    }
}
