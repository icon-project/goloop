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

package foundation.icon.test.cases;

import foundation.icon.icx.IconService;
import foundation.icon.icx.KeyWallet;
import foundation.icon.icx.data.Address;
import foundation.icon.icx.data.Bytes;
import foundation.icon.icx.transport.http.HttpProvider;
import foundation.icon.test.common.Constants;
import foundation.icon.test.common.Env;
import foundation.icon.test.common.TestBase;
import foundation.icon.test.common.TransactionHandler;
import foundation.icon.test.score.ContainerScore;
import org.junit.jupiter.api.AfterAll;
import org.junit.jupiter.api.BeforeAll;
import org.junit.jupiter.api.Tag;
import org.junit.jupiter.api.Test;

import java.math.BigInteger;
import java.util.ArrayList;
import java.util.Arrays;
import java.util.List;

import static foundation.icon.test.common.Env.LOG;
import static org.junit.jupiter.api.Assertions.assertArrayEquals;
import static org.junit.jupiter.api.Assertions.assertEquals;

@Tag(Constants.TAG_JAVA_GOV)
public class ContainerTest extends TestBase {
    private static TransactionHandler txHandler;
    private static KeyWallet[] wallets;
    private static KeyWallet ownerWallet;

    @BeforeAll
    static void setup() throws Exception {
        Env.Node node = Env.nodes[0];
        Env.Channel channel = node.channels[0];
        Env.Chain chain = channel.chain;
        IconService iconService = new IconService(new HttpProvider(channel.getAPIUrl(Env.testApiVer)));
        txHandler = new TransactionHandler(iconService, chain);

        // init wallets
        wallets = new KeyWallet[2];
        BigInteger amount = ICX.multiply(BigInteger.valueOf(100));
        for (int i = 0; i < wallets.length; i++) {
            wallets[i] = KeyWallet.create();
            txHandler.transfer(wallets[i].getAddress(), amount);
        }
        for (KeyWallet wallet : wallets) {
            ensureIcxBalance(txHandler, wallet.getAddress(), BigInteger.ZERO, amount);
        }
        ownerWallet = wallets[0];
    }

    @AfterAll
    static void destroy() throws Exception {
        for (KeyWallet wallet : wallets) {
            txHandler.refundAll(wallet);
        }
    }

    @Test
    void testPythonToJavaMigration() throws Exception {
        // deploy Python contract first
        ContainerScore testScore = ContainerScore.mustDeploy(txHandler, ownerWallet);

        KeyWallet alice = wallets[1];
        BigInteger[] intArray = new BigInteger[]{
                BigInteger.ZERO, BigInteger.ONE, ICX, ICX.multiply(ICX)
        };
        String[] strArray = new String[]{
                "", "Hello", "Good Morning"
        };
        byte[][] bytesArray = new byte[][]{
                strArray[1].getBytes(), strArray[2].getBytes()
        };
        Boolean[] boolArray = new Boolean[]{
                true, false
        };
        Address[] addrArray = new Address[]{
                ownerWallet.getAddress(), alice.getAddress(), testScore.getAddress()
        };

        // set some values to the contract
        setIntegers(testScore, intArray);
        setStrings(testScore, strArray);
        setBytes(testScore, bytesArray);
        setBools(testScore, boolArray);
        setAddresses(testScore, addrArray);

        // verify the values before migration
        verifyIntegers(testScore, intArray);
        verifyStrings(testScore, strArray);
        verifyBytes(testScore, bytesArray);
        verifyBools(testScore, boolArray);
        verifyAddresses(testScore, addrArray);

        // update to Java contract
        LOG.infoEntering("deploy", "update to Java SCORE");
        var hash = testScore.updateToJavaScore();
        assertSuccess(testScore.getResult(hash));
        LOG.infoExiting();

        // verify the values after migration
        verifyIntegers(testScore, intArray);
        verifyStrings(testScore, strArray);
        verifyBytes(testScore, bytesArray);
        verifyBools(testScore, boolArray);
        verifyAddresses(testScore, addrArray);
    }

    private void setIntegers(ContainerScore testScore, BigInteger[] intArray) throws Exception {
        List<Bytes> txes = new ArrayList<>();
        txes.add(testScore.setVar(BigInteger.valueOf(1000)));
        txes.add(testScore.setDict("icx", ICX));
        for (var item : intArray) {
            txes.add(testScore.setArray(item));
        }
        for (var tx : txes) {
            assertSuccess(txHandler.getResult(tx));
        }
    }

    private void verifyIntegers(ContainerScore testScore, BigInteger[] intArray) throws Exception {
        assertEquals(BigInteger.valueOf(1000), testScore.getVar(ContainerScore.T_INT));
        assertEquals(ICX, testScore.getDict("icx", ContainerScore.T_INT));
        var rpcList = testScore.getArray(ContainerScore.T_INT).asList();
        BigInteger[] result = new BigInteger[intArray.length];
        for (int i = 0; i < intArray.length; i++) {
            result[i] = rpcList.get(i).asInteger();
        }
        LOG.info("exp: " + Arrays.toString(intArray));
        LOG.info("ret: " + Arrays.toString(result));
        assertArrayEquals(intArray, result);
    }

    private void setStrings(ContainerScore testScore, String[] strArray) throws Exception {
        List<Bytes> txes = new ArrayList<>();
        txes.add(testScore.setVar("A"));
        txes.add(testScore.setDict("name", "Alice"));
        for (var item : strArray) {
            txes.add(testScore.setArray(item));
        }
        for (var tx : txes) {
            assertSuccess(txHandler.getResult(tx));
        }
    }

    private void verifyStrings(ContainerScore testScore, String[] strArray) throws Exception {
        assertEquals("A", testScore.getVar(ContainerScore.T_STRING));
        assertEquals("Alice", testScore.getDict("name", ContainerScore.T_STRING));
        var rpcList = testScore.getArray(ContainerScore.T_STRING).asList();
        String[] result = new String[strArray.length];
        for (int i = 0; i < strArray.length; i++) {
            result[i] = rpcList.get(i).asString();
        }
        LOG.info("exp: " + Arrays.toString(strArray));
        LOG.info("ret: " + Arrays.toString(result));
        assertArrayEquals(strArray, result);
    }

    private void setBytes(ContainerScore testScore, byte[][] bytesArray) throws Exception {
        List<Bytes> txes = new ArrayList<>();
        txes.add(testScore.setVar("A".getBytes()));
        txes.add(testScore.setDict("name", "Alice".getBytes()));
        for (var item : bytesArray) {
            txes.add(testScore.setArray(item));
        }
        for (var tx : txes) {
            assertSuccess(txHandler.getResult(tx));
        }
    }

    private void verifyBytes(ContainerScore testScore, byte[][] bytesArray) throws Exception {
        assertArrayEquals("A".getBytes(), (byte[]) testScore.getVar(ContainerScore.T_BYTES));
        assertArrayEquals("Alice".getBytes(), (byte[]) testScore.getDict("name", ContainerScore.T_BYTES));
        var rpcList = testScore.getArray(ContainerScore.T_BYTES).asList();
        byte[][] result = new byte[bytesArray.length][];
        for (int i = 0; i < bytesArray.length; i++) {
            result[i] = rpcList.get(i).asByteArray();
        }
        LOG.info("exp: " + Arrays.toString(bytesArray[0]));
        LOG.info("ret: " + Arrays.toString(result[0]));
        assertArrayEquals(bytesArray, result);
    }

    private void setBools(ContainerScore testScore, Boolean[] boolArray) throws Exception {
        List<Bytes> txes = new ArrayList<>();
        txes.add(testScore.setVar(boolArray[0]));
        txes.add(testScore.setDict("bool", boolArray[1]));
        for (var addr : boolArray) {
            txes.add(testScore.setArray(addr));
        }
        for (var tx : txes) {
            assertSuccess(txHandler.getResult(tx));
        }
    }

    private void verifyBools(ContainerScore testScore, Boolean[] boolArray) throws Exception {
        assertEquals(boolArray[0], testScore.getVar(ContainerScore.T_BOOL));
        assertEquals(boolArray[1], testScore.getDict("bool", ContainerScore.T_BOOL));
        var rpcList = testScore.getArray(ContainerScore.T_BOOL).asList();
        Boolean[] result = new Boolean[boolArray.length];
        for (int i = 0; i < boolArray.length; i++) {
            result[i] = rpcList.get(i).asBoolean();
        }
        LOG.info("exp: " + Arrays.toString(boolArray));
        LOG.info("ret: " + Arrays.toString(result));
        assertArrayEquals(boolArray, result);
    }

    private void setAddresses(ContainerScore testScore, Address[] addrArray) throws Exception {
        List<Bytes> txes = new ArrayList<>();
        txes.add(testScore.setVar(addrArray[0]));
        txes.add(testScore.setDict("alice", addrArray[1]));
        txes.add(testScore.setDict("self", addrArray[2]));
        for (var addr : addrArray) {
            txes.add(testScore.setArray(addr));
        }
        for (var tx : txes) {
            assertSuccess(txHandler.getResult(tx));
        }
    }

    private void verifyAddresses(ContainerScore testScore, Address[] addrArray) throws Exception {
        assertEquals(addrArray[0], testScore.getVar(ContainerScore.T_ADDRESS));
        assertEquals(addrArray[1], testScore.getDict("alice", ContainerScore.T_ADDRESS));
        assertEquals(addrArray[2], testScore.getDict("self", ContainerScore.T_ADDRESS));
        var rpcList = testScore.getArray(ContainerScore.T_ADDRESS).asList();
        Address[] result = new Address[addrArray.length];
        for (int i = 0; i < addrArray.length; i++) {
            result[i] = rpcList.get(i).asAddress();
        }
        LOG.info("exp: " + Arrays.toString(addrArray));
        LOG.info("ret: " + Arrays.toString(result));
        assertArrayEquals(addrArray, result);
    }
}
