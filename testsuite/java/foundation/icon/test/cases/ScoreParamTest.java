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

package foundation.icon.test.cases;

import foundation.icon.icx.IconService;
import foundation.icon.icx.KeyWallet;
import foundation.icon.icx.data.Address;
import foundation.icon.icx.data.Bytes;
import foundation.icon.icx.data.TransactionResult;
import foundation.icon.icx.transport.http.HttpProvider;
import foundation.icon.icx.transport.jsonrpc.RpcArray;
import foundation.icon.icx.transport.jsonrpc.RpcItem;
import foundation.icon.icx.transport.jsonrpc.RpcObject;
import foundation.icon.icx.transport.jsonrpc.RpcValue;
import foundation.icon.test.common.Constants;
import foundation.icon.test.common.Env;
import foundation.icon.test.common.TestBase;
import foundation.icon.test.common.TransactionHandler;
import foundation.icon.test.score.Score;
import org.junit.jupiter.api.BeforeAll;
import org.junit.jupiter.api.Tag;
import org.junit.jupiter.api.Test;

import java.math.BigInteger;
import java.util.ArrayList;
import java.util.List;

import static foundation.icon.test.common.Env.LOG;
import static org.junit.jupiter.api.Assertions.assertEquals;
import static org.junit.jupiter.api.Assertions.assertNull;
import static org.junit.jupiter.api.Assertions.assertTrue;

@Tag(Constants.TAG_PY_SCORE)
class ScoreParamTest extends TestBase {
    public static final String SCORE_CHECK_PARAMS_PATH = Score.getFilePath("check_params");

    private static TransactionHandler txHandler;
    private static KeyWallet callerWallet;
    private static Score testScore;
    private static Score interCallScore;

    private static final int TYPE_BOOL = 0;
    private static final int TYPE_ADDRESS = 1;
    private static final int TYPE_INT = 2;
    private static final int TYPE_BYTES = 3;
    private static final int TYPE_STR = 4;

    private static final String[] VALUES_FOR_STR = {
            "hello", "ZERO", "ONE",
            "0x0", "0x1", "0x12", "0xdd",
            "true", "false", "",
    };

    private static final byte[][] VALUES_FOR_BYTES = {
            {0x22, 0x33, 0x7f},
            {0}, {1}, "Hello".getBytes(), {},
    };

    private static final BigInteger[] VALUES_FOR_INT = {
            BigInteger.ONE, BigInteger.ZERO,
            BigInteger.valueOf(-1),
            BigInteger.valueOf(0x1FFFFFFFFL),
            new BigInteger("1FFFFFFFFFFFFFFFF", 16),
    };

    private static final boolean[] VALUES_FOR_BOOL = {true, false};

    private static final Address[] VALUES_FOR_ADDRESS = {
            new Address("cxd2a525388459fab5f3107e230c9868d118b8d15d"),
            new Address("hxb37a4fc334b472e4b13d5d67087deaab9a85a324"),
            new Address("cx0000000000000000000000000000000000000000"),
            new Address("hx0000000000000000000000000000000000000000"),
    };

    @BeforeAll
    static void init() throws Exception {
        Env.Node node = Env.nodes[0];
        Env.Channel channel = node.channels[0];
        Env.Chain chain = channel.chain;
        IconService iconService = new IconService(new HttpProvider(channel.getAPIUrl(Env.testApiVer)));
        txHandler = new TransactionHandler(iconService, chain);
        callerWallet = KeyWallet.create();

        KeyWallet ownerWallet = KeyWallet.create();
        testScore = txHandler.deploy(ownerWallet, SCORE_CHECK_PARAMS_PATH, null);
        interCallScore = txHandler.deploy(ownerWallet, SCORE_CHECK_PARAMS_PATH, null);
    }

    @Test
    void callInt() throws Exception {
        LOG.infoEntering("callInt");
        Bytes[] hashes = new Bytes[VALUES_FOR_INT.length];
        int cnt = 0;
        for (BigInteger p : VALUES_FOR_INT) {
            RpcObject params = new RpcObject.Builder()
                    .put("param", new RpcValue(p))
                    .build();
            LOG.infoEntering("invoke", p.toString());
            hashes[cnt++] = testScore.invoke(callerWallet, "call_int", params);
            LOG.infoExiting();
        }
        for (int i = 0; i < cnt; i++) {
            String expected = new RpcValue(VALUES_FOR_INT[i]).toString();
            LOG.infoEntering("check", "exp={" + expected + "}");
            TransactionResult result = txHandler.getResult(hashes[i]);
            assertSuccess(result);

            boolean checked = false;
            for (TransactionResult.EventLog el : result.getEventLogs()) {
                RpcItem sig = el.getIndexed().get(0);
                if (!sig.asString().equals("LogCallValue(bool,int,str,Address,bytes)")) {
                    continue;
                }
                RpcItem val = el.getData().get(1);
                assertEquals(expected, val.asString());
                checked = true;
            }
            assertTrue(checked);
            LOG.infoExiting();
        }
        LOG.infoExiting();
    }

    @Test
    void callStr() throws Exception {
        LOG.infoEntering("callStr");
        Bytes[] hashes = new Bytes[VALUES_FOR_STR.length];
        int cnt = 0;
        for (String p : VALUES_FOR_STR) {
            RpcObject params = new RpcObject.Builder()
                    .put("param", new RpcValue(p))
                    .build();
            LOG.infoEntering("invoke", p);
            hashes[cnt++] = testScore.invoke(callerWallet, "call_str", params);
            LOG.infoExiting();
        }
        for (int i = 0; i < cnt; i++) {
            String expected = VALUES_FOR_STR[i];
            LOG.infoEntering("check", "exp={" + expected + "}");
            TransactionResult result = txHandler.getResult(hashes[i]);
            assertSuccess(result);

            boolean checked = false;
            for (TransactionResult.EventLog el : result.getEventLogs()) {
                RpcItem sig = el.getIndexed().get(0);
                if (!sig.asString().equals("LogCallValue(bool,int,str,Address,bytes)")) {
                    continue;
                }
                RpcItem val = el.getData().get(2);
                assertEquals(expected, val.asString());
                checked = true;
            }
            assertTrue(checked);
            LOG.infoExiting();
        }
        LOG.infoExiting();
    }

    @Test
    void callBytes() throws Exception {
        LOG.infoEntering("callBytes");
        Bytes[] hashes = new Bytes[VALUES_FOR_BYTES.length];
        int cnt = 0;
        for (byte[] p : VALUES_FOR_BYTES) {
            RpcValue pv = new RpcValue(p);
            RpcObject params = new RpcObject.Builder()
                    .put("param", pv)
                    .build();
            LOG.infoEntering("invoke", pv.asString());
            hashes[cnt++] = testScore.invoke(callerWallet, "call_bytes", params);
            LOG.infoExiting();
        }
        for (int i = 0; i < cnt; i++) {
            String expected = new Bytes(VALUES_FOR_BYTES[i]).toString();
            LOG.infoEntering("check", "exp={" + expected + "}");
            TransactionResult result = txHandler.getResult(hashes[i]);
            assertSuccess(result);

            boolean checked = false;
            for (TransactionResult.EventLog el : result.getEventLogs()) {
                RpcItem sig = el.getIndexed().get(0);
                if (!sig.asString().equals("LogCallValue(bool,int,str,Address,bytes)")) {
                    continue;
                }
                RpcItem val = el.getData().get(4);
                assertEquals(expected, val.asString());
                checked = true;
            }
            assertTrue(checked);
            LOG.infoExiting();
        }
        LOG.infoExiting();
    }

    @Test
    void callBool() throws Exception {
        LOG.infoEntering("callBool");
        for (boolean p : VALUES_FOR_BOOL) {
            RpcObject params = new RpcObject.Builder()
                    .put("param", new RpcValue(p))
                    .build();
            LOG.infoEntering("invoke", String.valueOf(p));
            TransactionResult result =
                    testScore.invokeAndWaitResult(callerWallet, "call_bool",
                            params, BigInteger.valueOf(0), BigInteger.valueOf(100));
            LOG.infoExiting();
            assertEquals(Constants.STATUS_SUCCESS, result.getStatus());
            RpcItem item = testScore.call("check_bool", null);
            assertEquals(String.valueOf(p), item.asString());
        }
        LOG.infoExiting();
    }

    @Test
    void callAddress() throws Exception {
        LOG.infoEntering("callAddress");
        Bytes[] hashes = new Bytes[VALUES_FOR_ADDRESS.length];
        int cnt = 0;
        for (Address p : VALUES_FOR_ADDRESS) {
            RpcObject params = new RpcObject.Builder()
                    .put("param", new RpcValue(p))
                    .build();
            LOG.infoEntering("invoke", p.toString());
            hashes[cnt++] = testScore.invoke(callerWallet, "call_address", params);
            LOG.infoExiting();
        }
        for (int i = 0; i < cnt; i++) {
            String expected = VALUES_FOR_ADDRESS[i].toString();
            LOG.infoEntering("check", "exp={" + expected + "}");
            TransactionResult result = txHandler.getResult(hashes[i]);
            assertSuccess(result);

            boolean checked = false;
            for (TransactionResult.EventLog el : result.getEventLogs()) {
                RpcItem sig = el.getIndexed().get(0);
                if (!sig.asString().equals("LogCallValue(bool,int,str,Address,bytes)")) {
                    continue;
                }
                RpcItem val = el.getData().get(3);
                assertEquals(expected, val.asString());
                checked = true;
            }
            assertTrue(checked);
            LOG.infoExiting();
        }
        LOG.infoExiting();
    }

    @Test
    void callAll() throws Exception {
        LOG.infoEntering("callAll");
        RpcObject params = new RpcObject.Builder()
                .put("p_bool", new RpcValue(true))
                .put("p_addr", new RpcValue(KeyWallet.create().getAddress()))
                .put("p_int", new RpcValue(BigInteger.ONE))
                .put("p_str", new RpcValue("HELLO"))
                .put("p_bytes", new RpcValue(new byte[]{0x12}))
                .build();
        LOG.infoEntering("invoke call_all");
        TransactionResult result =
                testScore.invokeAndWaitResult(callerWallet, "call_all",
                        params, BigInteger.valueOf(0), BigInteger.valueOf(100));
        LOG.infoExiting();
        assertEquals(Constants.STATUS_SUCCESS, result.getStatus());
        RpcItem item = testScore.call("check_all", null);
        assertEquals("all", item.asString());
        LOG.infoExiting();
    }

    static class Person {
        String name;
        BigInteger age;

        public Person(String name, BigInteger age) {
            this.name = name;
            this.age = age;
        }

        @Override
        public boolean equals(Object obj) {
            if (obj instanceof Person) {
                Person other = (Person) obj;
                return (this.name.equals(other.name)
                        && this.age.equals(other.age));
            }
            return false;
        }
    }

    @Test
    void callStruct() throws Exception {
        LOG.infoEntering("callStruct");
        Person alice = new Person("Alice", BigInteger.valueOf(20));
        RpcObject params = new RpcObject.Builder()
                .put("person", new RpcObject.Builder()
                        .put("name", new RpcValue(alice.name))
                        .put("age", new RpcValue(alice.age))
                        .build()
                ).build();
        LOG.infoEntering("invoke call_struct");
        assertSuccess(testScore.invokeAndWaitResult(callerWallet,
                "call_struct", params, BigInteger.ZERO, BigInteger.valueOf(100)));
        LOG.infoExiting();
        RpcItem item = testScore.call("check_struct", null);
        Person other = new Person(
                item.asObject().getItem("name").asString(),
                item.asObject().getItem("age").asInteger());
        assertEquals(alice, other);
        LOG.infoExiting();
    }

    @Test
    void callListStruct() throws Exception {
        LOG.infoEntering("callListStruct");
        var peopleList = List.of(
                new Person("Alice", BigInteger.valueOf(20)),
                new Person("Bob", BigInteger.valueOf(30)),
                new Person("Charlie", BigInteger.valueOf(40))
        );
        var array = new RpcArray.Builder();
        for (var p : peopleList) {
            array.add(new RpcObject.Builder()
                    .put("name", new RpcValue(p.name))
                    .put("age", new RpcValue(p.age))
                    .build());
        }
        RpcObject params = new RpcObject.Builder()
                .put("people", array.build())
                .build();
        LOG.infoEntering("invoke call_list_struct");
        assertSuccess(testScore.invokeAndWaitResult(callerWallet,
                "call_list_struct", params, BigInteger.ZERO, BigInteger.valueOf(100)));
        LOG.infoExiting();
        RpcItem ret = testScore.call("check_list_struct", null);
        var retList = new ArrayList<Person>();
        for (RpcItem item : ret.asArray().asList()) {
            Person p = new Person(
                    item.asObject().getItem("name").asString(),
                    item.asObject().getItem("age").asInteger());
            retList.add(p);
        }
        assertEquals(peopleList, retList);
        LOG.infoExiting();
    }

    @Test
    void interCallBool() throws Exception {
        LOG.infoEntering("interCallBool");
        for (boolean p : VALUES_FOR_BOOL) {
            RpcObject params = new RpcObject.Builder()
                    .put("_to", new RpcValue(interCallScore.getAddress()))
                    .put("param", new RpcValue(p))
                    .put("ptype", new RpcValue(BigInteger.valueOf(TYPE_BOOL)))
                    .build();
            LOG.infoEntering("invoke", String.valueOf(p));
            TransactionResult result =
                    testScore.invokeAndWaitResult(callerWallet, "inter_call_bool",
                            params, BigInteger.valueOf(0), BigInteger.valueOf(100));
            LOG.infoExiting();
            assertEquals(Constants.STATUS_SUCCESS, result.getStatus());
            RpcItem item = interCallScore.call("check_bool", null);
            assertEquals(String.valueOf(p), item.asString());
        }
        LOG.infoExiting();
    }

    @Test
    void interCallAddress() throws Exception {
        LOG.infoEntering("interCallAddress");
        Bytes[] hashes = new Bytes[VALUES_FOR_ADDRESS.length];
        int cnt = 0;
        for (Address p : VALUES_FOR_ADDRESS) {
            RpcObject params = new RpcObject.Builder()
                    .put("_to", new RpcValue(interCallScore.getAddress()))
                    .put("param", new RpcValue(p))
                    .put("ptype", new RpcValue(BigInteger.valueOf(TYPE_ADDRESS)))
                    .build();
            LOG.infoEntering("invoke", p.toString());
            hashes[cnt++] = testScore.invoke(callerWallet, "inter_call_address", params);
            LOG.infoExiting();
        }
        for (int i = 0; i < cnt; i++) {
            String expected = VALUES_FOR_ADDRESS[i].toString();
            LOG.infoEntering("check", "exp={" + expected + "}");
            TransactionResult result = txHandler.getResult(hashes[i]);
            assertSuccess(result);

            boolean checked = false;
            for (TransactionResult.EventLog el : result.getEventLogs()) {
                RpcItem sig = el.getIndexed().get(0);
                Address scoreAddr = new Address(el.getScoreAddress());
                if (!sig.asString().equals("LogCallValue(bool,int,str,Address,bytes)")
                        || !scoreAddr.equals(interCallScore.getAddress())) {
                    continue;
                }
                RpcItem val = el.getData().get(3);
                assertEquals(expected, val.asString());
                checked = true;
            }
            assertTrue(checked);
            LOG.infoExiting();
        }
        LOG.infoExiting();
    }

    @Test
    void interCallInt() throws Exception {
        LOG.infoEntering("interCallInt");
        Bytes[] hashes = new Bytes[VALUES_FOR_INT.length];
        int cnt = 0;
        for (BigInteger p : VALUES_FOR_INT) {
            RpcObject params = new RpcObject.Builder()
                    .put("_to", new RpcValue(interCallScore.getAddress()))
                    .put("param", new RpcValue(p))
                    .put("ptype", new RpcValue(BigInteger.valueOf(TYPE_INT)))
                    .build();
            LOG.infoEntering("invoke", p.toString());
            hashes[cnt++] = testScore.invoke(callerWallet, "inter_call_int", params);
            LOG.infoExiting();
        }
        for (int i = 0; i < cnt; i++) {
            String expected = new RpcValue(VALUES_FOR_INT[i]).toString();
            LOG.infoEntering("check", "exp={" + expected + "}");
            TransactionResult result = txHandler.getResult(hashes[i]);
            assertSuccess(result);

            boolean checked = false;
            for (TransactionResult.EventLog el : result.getEventLogs()) {
                RpcItem sig = el.getIndexed().get(0);
                Address scoreAddr = new Address(el.getScoreAddress());
                if (!sig.asString().equals("LogCallValue(bool,int,str,Address,bytes)")
                        || !scoreAddr.equals(interCallScore.getAddress())) {
                    continue;
                }
                RpcItem val = el.getData().get(1);
                assertEquals(expected, val.asString());
                checked = true;
            }
            assertTrue(checked);
            LOG.infoExiting();
        }
        LOG.infoExiting();
    }

    @Test
    void interCallBytes() throws Exception {
        LOG.infoEntering("interCallBytes");
        Bytes[] hashes = new Bytes[VALUES_FOR_BYTES.length];
        int cnt = 0;
        for (byte[] p : VALUES_FOR_BYTES) {
            RpcValue pv = new RpcValue(p);
            RpcObject params = new RpcObject.Builder()
                    .put("_to", new RpcValue(interCallScore.getAddress()))
                    .put("param", pv)
                    .put("ptype", new RpcValue(BigInteger.valueOf(TYPE_BYTES)))
                    .build();
            LOG.infoEntering("invoke", pv.asString());
            hashes[cnt++] = testScore.invoke(callerWallet, "inter_call_bytes", params);
            LOG.infoExiting();
        }
        for (int i = 0; i < cnt; i++) {
            String expected = new Bytes(VALUES_FOR_BYTES[i]).toString();
            LOG.infoEntering("check", "exp={" + expected + "}");
            TransactionResult result = txHandler.getResult(hashes[i]);
            assertSuccess(result);

            boolean checked = false;
            for (TransactionResult.EventLog el : result.getEventLogs()) {
                RpcItem sig = el.getIndexed().get(0);
                Address scoreAddr = new Address(el.getScoreAddress());
                if (!sig.asString().equals("LogCallValue(bool,int,str,Address,bytes)")
                        || !scoreAddr.equals(interCallScore.getAddress())) {
                    continue;
                }
                RpcItem val = el.getData().get(4);
                assertEquals(expected, val.asString());
                checked = true;
            }
            assertTrue(checked);
            LOG.infoExiting();
        }
        LOG.infoExiting();
    }

    @Test
    void interCallStr() throws Exception {
        LOG.infoEntering("interCallStr");
        Bytes[] hashes = new Bytes[VALUES_FOR_STR.length];
        int cnt = 0;
        for (String p : VALUES_FOR_STR) {
            RpcObject params = new RpcObject.Builder()
                    .put("_to", new RpcValue(interCallScore.getAddress()))
                    .put("param", new RpcValue(p))
                    .put("ptype", new RpcValue(BigInteger.valueOf(TYPE_STR)))
                    .build();
            LOG.infoEntering("invoke", p);
            hashes[cnt++] = testScore.invoke(callerWallet, "inter_call_str", params);
            LOG.infoExiting();
        }
        for (int i = 0; i < cnt; i++) {
            String expected = VALUES_FOR_STR[i];
            LOG.infoEntering("check", "exp={" + expected + "}");
            TransactionResult result = txHandler.getResult(hashes[i]);
            assertSuccess(result);

            boolean checked = false;
            for (TransactionResult.EventLog el : result.getEventLogs()) {
                RpcItem sig = el.getIndexed().get(0);
                Address scoreAddr = new Address(el.getScoreAddress());
                if (!sig.asString().equals("LogCallValue(bool,int,str,Address,bytes)")
                        || !scoreAddr.equals(interCallScore.getAddress())) {
                    continue;
                }
                RpcItem val = el.getData().get(2);
                assertEquals(expected, val.asString());
                checked = true;
            }
            assertTrue(checked);
            LOG.infoExiting();
        }
        LOG.infoExiting();
    }

    @Test
    void interCallAll() throws Exception {
        LOG.infoEntering("interCallAll");
        RpcObject params = new RpcObject.Builder()
                .put("_to", new RpcValue(interCallScore.getAddress()))
                .put("p_bool", new RpcValue(true))
                .put("p_addr", new RpcValue(KeyWallet.create().getAddress()))
                .put("p_int", new RpcValue(BigInteger.ONE))
                .put("p_str", new RpcValue("HELLO"))
                .put("p_bytes", new RpcValue(new byte[]{0x12}))
                .build();
        LOG.infoEntering("invoke inter_call_all");
        TransactionResult result =
                testScore.invokeAndWaitResult(callerWallet, "inter_call_all",
                        params, BigInteger.valueOf(0), BigInteger.valueOf(100));
        LOG.infoExiting();
        assertEquals(Constants.STATUS_SUCCESS, result.getStatus());
        RpcItem item = interCallScore.call("check_all", null);
        assertEquals("all", item.asString());
        LOG.infoExiting();
    }

    @Test
    void interCallStruct() throws Exception {
        LOG.infoEntering("interCallStruct");
        Person alice = new Person("Alice1", BigInteger.valueOf(21));
        RpcObject params = new RpcObject.Builder()
                .put("_to", new RpcValue(interCallScore.getAddress()))
                .put("person", new RpcObject.Builder()
                        .put("name", new RpcValue(alice.name))
                        .put("age", new RpcValue(alice.age))
                        .build()
                ).build();
        LOG.infoEntering("invoke inter_call_struct");
        assertSuccess(testScore.invokeAndWaitResult(callerWallet,
                "inter_call_struct", params, BigInteger.ZERO, BigInteger.valueOf(100)));
        LOG.infoExiting();
        RpcItem item = interCallScore.call("check_struct", null);
        Person other = new Person(
                item.asObject().getItem("name").asString(),
                item.asObject().getItem("age").asInteger());
        assertEquals(alice, other);
        LOG.infoExiting();
    }

    @Test
    void interCallListStruct() throws Exception {
        LOG.infoEntering("interCallListStruct");
        var peopleList = List.of(
                new Person("Alice1", BigInteger.valueOf(21)),
                new Person("Bob1", BigInteger.valueOf(31)),
                new Person("Charlie1", BigInteger.valueOf(41))
        );
        var array = new RpcArray.Builder();
        for (var p : peopleList) {
            array.add(new RpcObject.Builder()
                    .put("name", new RpcValue(p.name))
                    .put("age", new RpcValue(p.age))
                    .build());
        }
        RpcObject params = new RpcObject.Builder()
                .put("_to", new RpcValue(interCallScore.getAddress()))
                .put("people", array.build())
                .build();
        LOG.infoEntering("invoke inter_call_list_struct");
        assertSuccess(testScore.invokeAndWaitResult(callerWallet,
                "inter_call_list_struct", params, BigInteger.ZERO, BigInteger.valueOf(100)));
        LOG.infoExiting();
        RpcItem ret = interCallScore.call("check_list_struct", null);
        var retList = new ArrayList<Person>();
        for (RpcItem item : ret.asArray().asList()) {
            Person p = new Person(
                    item.asObject().getItem("name").asString(),
                    item.asObject().getItem("age").asInteger());
            retList.add(p);
        }
        assertEquals(peopleList, retList);
        LOG.infoExiting();
    }

    @Test
    void invalidInterCallBool() throws Exception {
        LOG.infoEntering("invalidInterCallBool");
        Bytes[] hashes = new Bytes[4];
        int cnt = 0;
        for (int t : new int[]{TYPE_ADDRESS, TYPE_INT, TYPE_BYTES, TYPE_STR}) {
            RpcObject params = new RpcObject.Builder()
                    .put("_to", new RpcValue(interCallScore.getAddress()))
                    .put("param", new RpcValue(true))
                    .put("ptype", new RpcValue(BigInteger.valueOf(t)))
                    .build();
            LOG.infoEntering("invoke", String.valueOf(t));
            hashes[cnt++] = testScore.invoke(callerWallet, "inter_call_bool", params);
            LOG.infoExiting();
        }
        for (int i = 0; i < cnt; i++) {
            assertFailure(txHandler.getResult(hashes[i]));
        }
        LOG.infoExiting();
    }

    @Test
    void invalidInterCallAddress() throws Exception {
        LOG.infoEntering("invalidInterCallAddress");
        Bytes[] hashes = new Bytes[4];
        int cnt = 0;
        for (int t : new int[]{TYPE_BOOL, TYPE_INT, TYPE_BYTES, TYPE_STR}) {
            RpcObject params = new RpcObject.Builder()
                    .put("_to", new RpcValue(interCallScore.getAddress()))
                    .put("param", new RpcValue(KeyWallet.create().getAddress()))
                    .put("ptype", new RpcValue(BigInteger.valueOf(t)))
                    .build();
            LOG.infoEntering("invoke", String.valueOf(t));
            hashes[cnt++] = testScore.invoke(callerWallet, "inter_call_address", params);
            LOG.infoExiting();
        }
        for (int i = 0; i < cnt; i++) {
            assertFailure(txHandler.getResult(hashes[i]));
        }
        LOG.infoExiting();
    }

    @Test
    void invalidInterCallBytes() throws Exception {
        LOG.infoEntering("invalidInterCallBytes");
        Bytes[] hashes = new Bytes[4];
        int cnt = 0;
        for (int t : new int[]{TYPE_BOOL, TYPE_INT, TYPE_ADDRESS, TYPE_STR}) {
            RpcObject params = new RpcObject.Builder()
                    .put("_to", new RpcValue(interCallScore.getAddress()))
                    .put("param", new RpcValue(new byte[]{10}))
                    .put("ptype", new RpcValue(BigInteger.valueOf(t)))
                    .build();
            LOG.infoEntering("invoke", String.valueOf(t));
            hashes[cnt++] = testScore.invoke(callerWallet, "inter_call_bytes", params);
            LOG.infoExiting();
        }
        for (int i = 0; i < cnt; i++) {
            assertFailure(txHandler.getResult(hashes[i]));
        }
        LOG.infoExiting();
    }

    @Test
    void invalidInterCallStr() throws Exception {
        LOG.infoEntering("invalidInterCallStr");
        Bytes[] hashes = new Bytes[4];
        int cnt = 0;
        for (int t : new int[]{TYPE_BOOL, TYPE_INT, TYPE_ADDRESS, TYPE_BYTES}) {
            RpcObject params = new RpcObject.Builder()
                    .put("_to", new RpcValue(interCallScore.getAddress()))
                    .put("param", new RpcValue("HI"))
                    .put("ptype", new RpcValue(BigInteger.valueOf(t)))
                    .build();
            LOG.infoEntering("invoke", String.valueOf(t));
            hashes[cnt++] = testScore.invoke(callerWallet, "inter_call_str", params);
            LOG.infoExiting();
        }
        for (int i = 0; i < cnt; i++) {
            assertFailure(txHandler.getResult(hashes[i]));
        }
        LOG.infoExiting();
    }

    @Test
    void invalidInterCallInt() throws Exception {
        LOG.infoEntering("invalidInterCallInt");
        Bytes[] hashes = new Bytes[4];
        int cnt = 0;
        for (int t : new int[]{TYPE_BOOL, TYPE_BYTES, TYPE_ADDRESS, TYPE_STR}) {
            RpcObject params = new RpcObject.Builder()
                    .put("_to", new RpcValue(interCallScore.getAddress()))
                    .put("param", new RpcValue(BigInteger.ONE))
                    .put("ptype", new RpcValue(BigInteger.valueOf(t)))
                    .build();
            LOG.infoEntering("invoke", String.valueOf(t));
            hashes[cnt++] = testScore.invoke(callerWallet, "inter_call_int", params);
            LOG.infoExiting();
        }
        for (int i = 0; i < cnt; i++) {
            assertFailure(txHandler.getResult(hashes[i]));
        }
        LOG.infoExiting();
    }

    @Test
    void callDefaultParam() throws Exception {
        LOG.infoEntering("callDefaultParam");
        String param = "Hello";
        RpcObject params = new RpcObject.Builder()
                .put("default_param", new RpcValue(param.getBytes()))
                .build();
        LOG.infoEntering("invoke", param);
        TransactionResult result =
                testScore.invokeAndWaitResult(callerWallet, "call_default_param",
                        params, BigInteger.valueOf(0), BigInteger.valueOf(100));
        LOG.infoExiting();
        assertEquals(Constants.STATUS_SUCCESS, result.getStatus());
        RpcItem item = testScore.call("check_default", null);
        assertEquals(param, item.asString());

        params = new RpcObject.Builder()
                .build();
        LOG.infoEntering("invoke", "without param");
        result = testScore.invokeAndWaitResult(callerWallet, "call_default_param",
                        params, BigInteger.valueOf(0), BigInteger.valueOf(100));
        LOG.infoExiting();
        assertEquals(Constants.STATUS_SUCCESS, result.getStatus());
        item = testScore.call("check_default", null);
        assertEquals("None", item.asString());
        LOG.infoExiting();
    }

    @Test
    void interCallDefaultParam() throws Exception {
        LOG.infoEntering("interCallDefaultParam");
        String param = "Hello";
        RpcObject params = new RpcObject.Builder()
                .put("default_param", new RpcValue(param.getBytes()))
                .build();
        LOG.infoEntering("invoke", param);
        TransactionResult result =
                interCallScore.invokeAndWaitResult(callerWallet, "call_default_param",
                        params, BigInteger.valueOf(0), BigInteger.valueOf(100));
        LOG.infoExiting();
        assertEquals(Constants.STATUS_SUCCESS, result.getStatus());
        RpcItem item = interCallScore.call("check_default", null);
        assertEquals(param, item.asString());

        params = new RpcObject.Builder()
                .put("_to", new RpcValue(interCallScore.getAddress()))
                .build();
        LOG.infoEntering("invoke inter_call_default_param");
        result = testScore.invokeAndWaitResult(callerWallet, "inter_call_default_param",
                        params, BigInteger.valueOf(0), BigInteger.valueOf(100));
        LOG.infoExiting();
        assertEquals(Constants.STATUS_SUCCESS, result.getStatus());
        item = interCallScore.call("check_default", null);
        assertEquals("None", item.asString());
        LOG.infoExiting();
    }

    @Test
    void interCallWithNull() throws Exception {
        LOG.infoEntering("interCallWithNull");
        Bytes[] hashes = new Bytes[5];
        int cnt = 0;
        for (int t : new int[]{TYPE_BOOL, TYPE_ADDRESS, TYPE_INT, TYPE_BYTES, TYPE_STR}) {
            RpcObject params = new RpcObject.Builder()
                    .put("_to", new RpcValue(interCallScore.getAddress()))
                    .put("ptype", new RpcValue(BigInteger.valueOf(t)))
                    .build();
            LOG.infoEntering("invoke", String.valueOf(t));
            hashes[cnt++] = testScore.invoke(callerWallet, "inter_call_with_none", params);
            LOG.infoExiting();
        }
        for (int i = 0; i < cnt; i++) {
            assertFailure(txHandler.getResult(hashes[i]));
        }
        LOG.infoExiting();
    }

    @Test
    void interCallWithMoreParams() throws Exception {
        LOG.infoEntering("interCallWithMore");
        RpcObject params = new RpcObject.Builder()
                .put("_to", new RpcValue(interCallScore.getAddress()))
                .build();
        LOG.infoEntering("invoke inter_call_with_more_params");
        TransactionResult result =
                testScore.invokeAndWaitResult(callerWallet, "inter_call_with_more_params",
                        params, BigInteger.valueOf(0), BigInteger.valueOf(100));
        LOG.infoExiting();
        assertFailure(result);
        LOG.infoExiting();
    }

    @Test
    void invalidAddUndefinedParam() throws Exception {
        LOG.infoEntering("invalidAddUndefinedParam");
        RpcObject params = new RpcObject.Builder()
                .put("undefined1", new RpcValue(true))
                .put("undefined2", new RpcValue(BigInteger.ONE))
                .build();
        LOG.infoEntering("invoke call_default_param");
        TransactionResult result =
                testScore.invokeAndWaitResult(callerWallet, "call_default_param",
                        params, BigInteger.valueOf(0), BigInteger.valueOf(100));
        LOG.infoExiting();
        assertFailure(result);
        LOG.infoExiting();
    }

    @Test
    void interCallWithEmptyString() throws Exception {
        LOG.infoEntering("interCallWithEmptyString");
        RpcObject params = new RpcObject.Builder()
                .put("_to", new RpcValue(interCallScore.getAddress()))
                .build();
        LOG.infoEntering("invoke inter_call_empty_str");
        TransactionResult result =
                testScore.invokeAndWaitResult(callerWallet, "inter_call_empty_str",
                        params, BigInteger.valueOf(0), BigInteger.valueOf(100));
        LOG.infoExiting();
        assertEquals(Constants.STATUS_SUCCESS, result.getStatus());
        RpcItem item = interCallScore.call("check_str", null);
        assertEquals("", item.asString());
        LOG.infoExiting();
    }

    @Test
    void interCallWithDefaultParam() throws Exception {
        LOG.infoEntering("interCallWithDefaultParam");
        RpcObject params = new RpcObject.Builder()
                .put("_to", new RpcValue(interCallScore.getAddress()))
                .build();
        LOG.infoEntering("invoke inter_call_with_default_param");
        TransactionResult result =
                testScore.invokeAndWaitResult(callerWallet, "inter_call_with_default_param",
                        params, BigInteger.valueOf(0), BigInteger.valueOf(100));
        LOG.infoExiting();
        assertEquals(Constants.STATUS_SUCCESS, result.getStatus());
        // check the saved values
        RpcItem item = interCallScore.call("check_bool", null);
        assertEquals("true", item.asString());
        item = interCallScore.call("check_address", null);
        assertEquals("cx0000000000000000000000000000000000000000", item.asString());
        item = interCallScore.call("check_int", null);
        assertEquals("0", item.asString());
        item = interCallScore.call("check_str", null);
        assertEquals("", item.asString());
        item = interCallScore.call("check_bytes", null);
        assertEquals("0x00", item.asString());
        LOG.infoExiting();
    }

    @Test
    void checkSender() throws Exception {
        LOG.infoEntering("checkSender");
        RpcItem item = testScore.call("check_sender", null);
        assertNull(item);
        LOG.infoExiting();
    }

    @Test
    void callAllDefault() throws Exception {
        final int NUM = 5;
        final int CASES = 1 << NUM;

        LOG.infoEntering("callAllDefault");
        String[] names = {"_bool", "_int", "_str", "_addr", "_bytes"};
        RpcValue[] values = {
                new RpcValue(VALUES_FOR_BOOL[0]),
                new RpcValue(VALUES_FOR_INT[0]),
                new RpcValue(VALUES_FOR_STR[0]),
                new RpcValue(VALUES_FOR_ADDRESS[0]),
                new RpcValue(VALUES_FOR_BYTES[0]),
        };

        LOG.infoEntering("sending transactions");
        Bytes[] ids = new Bytes[CASES];
        for (int i = 0; i < CASES; i++) {
            RpcObject.Builder pb = new RpcObject.Builder();
            for (int idx = 0; idx < NUM; idx++) {
                if ((i & (1 << idx)) != 0) {
                    pb.put(names[idx], values[idx]);
                }
            }
            RpcObject params = pb.build();
            LOG.info("case=" + i + " sending tx param=" + params.toString());
            ids[i] = testScore.invoke(callerWallet, "call_all_default", params);
            LOG.info("txid=" + ids[i].toString());
        }
        LOG.infoExiting();

        for (int i = 0; i < CASES; i++) {
            LOG.infoEntering("checking", "case=" + i + " txid=" + ids[i]);

            TransactionResult result = testScore.getResult(ids[i]);
            assertEquals(result.getStatus(), Constants.STATUS_SUCCESS);

            boolean checked = false;
            for (TransactionResult.EventLog el : result.getEventLogs()) {
                RpcItem sig = el.getIndexed().get(0);
                if (!sig.asString().equals("LogCallValue(bool,int,str,Address,bytes)")) {
                    continue;
                }
                for (int idx = 0; idx < NUM; idx++) {
                    RpcItem val = el.getData().get(idx);
                    if ((i & (1 << idx)) != 0) {
                        assertEquals(values[idx].asString(), val.asString());
                    } else {
                        assertTrue(val.isNull());
                    }
                }
                checked = true;
            }
            assertTrue(checked);
            LOG.infoExiting();
        }
        LOG.infoExiting();
    }

    @Test
    void interCallWithLessParams() throws Exception {
        LOG.infoEntering("interCallWithLessParams");

        RpcObject params = new RpcObject.Builder()
                .put("_to", new RpcValue(interCallScore.getAddress()))
                .put("_bool", new RpcValue(true))
                .put("_int", new RpcValue(BigInteger.TEN))
                .build();

        TransactionResult result = testScore.invokeAndWaitResult(
                callerWallet, "inter_call_with_less_params", params);
        assertEquals(result.getStatus(), Constants.STATUS_SUCCESS);

        boolean checked = false;
        for (TransactionResult.EventLog el : result.getEventLogs()) {
            if (!interCallScore.getAddress().toString().equals(el.getScoreAddress())) {
                continue;
            }
            RpcItem sig = el.getIndexed().get(0);
            if (!sig.asString().equals("LogCallValue(bool,int,str,Address,bytes)")) {
                continue;
            }
            assertEquals("0x1", el.getData().get(0).asString());
            assertEquals("0xa", el.getData().get(1).asString());
            assertTrue(el.getData().get(2).isNull());
            assertTrue(el.getData().get(3).isNull());
            assertTrue(el.getData().get(4).isNull());
            checked = true;
        }
        assertTrue(checked);

        LOG.infoExiting();
    }
}
