package collection;

import avm.*;

public class CollectionTest
{
    static boolean equals(String ob, String exp) {
        if (ob==exp) {
            return true;
        }
        if (ob==null || exp==null) {
            return false;
        }
        return ob.equals(exp);
    }

    static void expectEquals(String ob, String exp) {
        if (equals(ob, exp)) {
            Blockchain.println("OK: observed:" + ob);
        } else {
            Blockchain.println("ERROR: observed:" + ob + " expected:" + exp);
        }
    }

    static void expectEquals(int ob, int exp) {
        if (ob==exp) {
            Blockchain.println("OK: observed:" + ob);
        } else {
            Blockchain.println("ERROR: observed:" + ob + " expected:" + exp);
        }
    }

    public static void onInstall() {
        String s;

        VarDB<String> vdb = Blockchain.<String>newVarDB("vdb");
        vdb.set("test");
        s = vdb.get();
        expectEquals(s, "test");

        DictDB<Integer, String> ddb = Blockchain.<Integer, String>newDictDB("ddb");
        ddb.set(10, "10");
        ddb.set(20, "20");
        s = ddb.get(10);
        expectEquals(s, "10");
        s = ddb.get(20);
        expectEquals(s, "20");

        ArrayDB<String> adb = Blockchain.<String>newArrayDB("adb");
        adb.add("0");
        adb.add("1");
        adb.add("2");
        expectEquals(adb.size(), 3);
        s = adb.get(0);
        expectEquals(s, "0");
        s = adb.get(1);
        expectEquals(s, "1");
        s = adb.get(2);
        expectEquals(s, "2");
        s = adb.pop();
        expectEquals(s, "2");
        s = adb.pop();
        expectEquals(s, "1");
        s = adb.pop();
        expectEquals(s, "0");
        expectEquals(adb.size(), 0);

        var dddb = Blockchain.<Integer, DictDB<Integer, String>>newDictDB("dddb");
        dddb.at(0).set(1, "0, 1");
        dddb.at(1).set(2, "1, 2");
        s = dddb.at(0).get(1);
        expectEquals(s, "0, 1");
        s = dddb.at(1).get(2);
        expectEquals(s, "1, 2");

        var dadb = Blockchain.<Integer, ArrayDB<String>>newDictDB("dadb");
        dadb.at(0).add("a0");
        dadb.at(0).add("a1");
        s = dadb.at(0).get(0);
        expectEquals(s, "a0");
        s = dadb.at(0).get(1);
        expectEquals(s, "a1");
        dadb.at(0).pop();
        dadb.at(0).pop();
        expectEquals(dadb.at(0).size(), 0);
    }
}
