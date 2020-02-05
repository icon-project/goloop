/*
 * Copyright 2019 ICON Foundation
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

package collection;

import avm.*;
import foundation.icon.ee.tooling.abi.External;

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

    public CollectionTest() {
        String s;

        VarDB<String> vdb = Blockchain.newVarDB("vdb", String.class);
        vdb.set("test");
        s = vdb.get();
        expectEquals(s, "test");

        DictDB<Integer, String> ddb = Blockchain.newDictDB("ddb", String.class);
        ddb.set(10, "10");
        ddb.set(20, "20");
        s = ddb.get(10);
        expectEquals(s, "10");
        s = ddb.get(20);
        expectEquals(s, "20");

        ArrayDB<String> adb = Blockchain.newArrayDB("adb", String.class);
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

        NestingDictDB<Integer, DictDB<Integer, String>> dddb = Blockchain.newNestingDictDB("dddb", String.class);
        dddb.at(0).set(1, "0, 1");
        dddb.at(1).set(2, "1, 2");
        s = dddb.at(0).get(1);
        expectEquals(s, "0, 1");
        s = dddb.at(1).get(2);
        expectEquals(s, "1, 2");

        NestingDictDB<Integer, ArrayDB<String>> dadb = Blockchain.newNestingDictDB("dadb", String.class);
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

    @External
    public int getInt() {
        return 11;
    }

    private static Address sampleTokenAddress() {
        var ba = new BigInteger("784b61a531e819838e1f308287f953015020000a", 16).toByteArray();
        var ba2 = new byte[ba.length+1];
        System.arraycopy(ba, 0, ba2, 1, ba.length);
        ba2[0] = 1;
        return new Address(ba2);
    }

    @External
    public BigInteger totalSupply2(Address sc) {
        return (BigInteger)Blockchain.call(sc, "totalSupply");
    }

    @External
    public BigInteger balanceOf2(Address sc, Address _owner) {
        return (BigInteger) Blockchain.call(sc, "balanceOf", _owner);
    }
}
