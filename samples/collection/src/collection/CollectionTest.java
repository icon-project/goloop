package collection;

import avm.*;

import foundation.icon.ee.tooling.abi.EventLog;
import foundation.icon.ee.tooling.abi.External;
import foundation.icon.ee.tooling.abi.Optional;
import foundation.icon.ee.tooling.abi.Payable;

import java.math.BigInteger;

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
        vdb.putValue("test");
        s = vdb.getValue();
        expectEquals(s, "test");

        DictDB<Integer, String> ddb = Blockchain.<Integer, String>newDictDB("ddb");
        ddb.putValue(10, "10");
        ddb.putValue(20, "20");
        s = ddb.getValue(10);
        expectEquals(s, "10");
        s = ddb.getValue(20);
        expectEquals(s, "20");

        ArrayDB<String> adb = Blockchain.<String>newArrayDB("adb");
        adb.addValue("0");
        adb.addValue("1");
        adb.addValue("2");
        expectEquals(adb.size(), 3);
        s = adb.getValue(0);
        expectEquals(s, "0");
        s = adb.getValue(1);
        expectEquals(s, "1");
        s = adb.getValue(2);
        expectEquals(s, "2");
        s = adb.popValue();
        expectEquals(s, "2");
        s = adb.popValue();
        expectEquals(s, "1");
        s = adb.popValue();
        expectEquals(s, "0");
        expectEquals(adb.size(), 0);

        var dddb = Blockchain.<Integer, DictDB<Integer, String>>newDictDB("dddb");
        dddb.get(0).putValue(1, "0, 1");
        dddb.get(1).putValue(2, "1, 2");
        s = dddb.get(0).getValue(1);
        expectEquals(s, "0, 1");
        s = dddb.get(1).getValue(2);
        expectEquals(s, "1, 2");

        var dadb = Blockchain.<Integer, ArrayDB<String>>newDictDB("dadb");
        dadb.get(0).addValue("a0");
        dadb.get(0).addValue("a1");
        s = dadb.get(0).getValue(0);
        expectEquals(s, "a0");
        s = dadb.get(0).getValue(1);
        expectEquals(s, "a1");
        dadb.get(0).popValue();
        dadb.get(0).popValue();
        expectEquals(dadb.get(0).size(), 0);
    }
}
