package foundation.icon.ee.ipc;

import foundation.icon.ee.test.GoldenTest;
import foundation.icon.ee.tooling.abi.External;
import org.junit.jupiter.api.Test;
import score.ArrayDB;
import score.Context;
import score.DictDB;
import score.NestingDictDB;
import score.VarDB;

public class QueryTest extends GoldenTest {
    public static class Score {
        @External(readonly=true)
        public int setDictDB() {
            DictDB<String , String> ddb = Context.newDictDB("ddb", String.class);
            try {
                ddb.set("key", "value");
                Context.println("unexpected");
            } catch (IllegalStateException e) {
                Context.println("OK: " + e);
            }
            return 0;
        }

        @External(readonly=true)
        public int setNestingDictDB() {
            NestingDictDB<String, DictDB<String , String>> ddb
                    = Context.newNestingDictDB("ddb", String.class);
            try {
                ddb.at("key").set("key", "value");
                Context.println("unexpected");
            } catch (IllegalStateException e) {
                Context.println("OK: " + e);
            }
            return 0;
        }

        @External(readonly=true)
        public int setArrayDB() {
            ArrayDB<String> adb = Context.newArrayDB("adb", String.class);
            try {
                adb.add("value");
                Context.println("unexpected");
            } catch (IllegalStateException e) {
                Context.println("OK: " + e);
            }
            return 0;
        }

        @External(readonly=true)
        public int setVarDB() {
            VarDB<String> vdb = Context.newVarDB("vdb", String.class);
            try {
                vdb.set("value");
                Context.println("unexpected");
            } catch (IllegalStateException e) {
                Context.println("OK: " + e);
            }
            return 0;
        }
    }

    @Test
    void testSetDB() {
        var score = sm.deploy(Score.class);
        score.query("setDictDB");
        score.query("setNestingDictDB");
        score.query("setArrayDB");
        score.query("setVarDB");
    }
}
