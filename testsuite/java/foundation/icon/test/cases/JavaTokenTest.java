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

package foundation.icon.test.cases;

import example.IRC2BasicToken;
import example.IRC3BasicToken;
import example.token.IRC2;
import example.token.IRC2Basic;
import example.token.IRC3;
import example.token.IRC3Basic;
import example.util.EnumerableIntMap;
import example.util.EnumerableSet;
import example.util.IntSet;
import foundation.icon.icx.IconService;
import foundation.icon.icx.KeyWallet;
import foundation.icon.icx.data.Address;
import foundation.icon.icx.data.Bytes;
import foundation.icon.icx.data.TransactionResult;
import foundation.icon.icx.transport.http.HttpProvider;
import foundation.icon.icx.transport.jsonrpc.RpcError;
import foundation.icon.test.common.Constants;
import foundation.icon.test.common.Env;
import foundation.icon.test.common.TestBase;
import foundation.icon.test.common.TransactionHandler;
import foundation.icon.test.score.IRC3TokenScore;
import foundation.icon.test.score.SampleTokenScore;
import org.junit.jupiter.api.AfterAll;
import org.junit.jupiter.api.BeforeAll;
import org.junit.jupiter.api.Tag;
import org.junit.jupiter.api.Test;

import java.math.BigInteger;
import java.security.SecureRandom;

import static foundation.icon.test.common.Env.LOG;
import static org.junit.jupiter.api.Assertions.assertEquals;
import static org.junit.jupiter.api.Assertions.assertThrows;

@Tag(Constants.TAG_JAVA_SCORE)
public class JavaTokenTest extends TestBase {
    private static final Address ZERO_ADDRESS = new Address("hx0000000000000000000000000000000000000000");
    private static TransactionHandler txHandler;
    private static SecureRandom secureRandom;
    private static KeyWallet[] wallets;
    private static KeyWallet ownerWallet, caller;

    @BeforeAll
    static void init() throws Exception {
        Env.Node node = Env.nodes[0];
        Env.Channel channel = node.channels[0];
        Env.Chain chain = channel.chain;
        IconService iconService = new IconService(new HttpProvider(channel.getAPIUrl(Env.testApiVer)));
        txHandler = new TransactionHandler(iconService, chain);
        secureRandom = new SecureRandom();

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
        caller = wallets[1];
    }

    @AfterAll
    static void shutdown() throws Exception {
        for (KeyWallet wallet : wallets) {
            txHandler.refundAll(wallet);
        }
    }

    private byte[] getRandomBytes(int size) {
        byte[] bytes = new byte[size];
        secureRandom.nextBytes(bytes);
        bytes[0] = 0; // make positive
        return bytes;
    }

    @Test
    public void testSampleToken() throws Exception {
        // 1. deploy
        BigInteger decimals = BigInteger.valueOf(18);
        BigInteger initialSupply = BigInteger.valueOf(1000);
        SampleTokenScore tokenScore = SampleTokenScore.mustDeploy(txHandler, ownerWallet,
                decimals, initialSupply, Constants.CONTENT_TYPE_JAVA);
        startTokenTest(tokenScore, decimals, initialSupply);
    }

    @Test
    public void testIRC2Token() throws Exception {
        // 1. deploy
        BigInteger decimals = BigInteger.valueOf(18);
        BigInteger initialSupply = BigInteger.valueOf(1000);
        SampleTokenScore tokenScore = SampleTokenScore.mustDeploy(txHandler, ownerWallet,
                decimals, initialSupply, new Class<?>[]{IRC2BasicToken.class, IRC2Basic.class, IRC2.class});
        startTokenTest(tokenScore, decimals, initialSupply);
    }

    private void startTokenTest(SampleTokenScore tokenScore, BigInteger decimals, BigInteger initialSupply)
            throws Exception {
        KeyWallet calleeWallet = KeyWallet.create();

        // 2. balanceOf
        LOG.infoEntering("balanceOf", "owner (initial)");
        BigInteger oneToken = BigInteger.TEN.pow(decimals.intValue());
        BigInteger totalSupply = oneToken.multiply(initialSupply);
        BigInteger bal = tokenScore.balanceOf(ownerWallet.getAddress());
        LOG.info("expected (" + totalSupply + "), got (" + bal + ")");
        assertEquals(totalSupply, bal);
        LOG.infoExiting();

        // 3. transfer #1
        LOG.infoEntering("transfer", "#1");
        TransactionResult result = tokenScore.transfer(ownerWallet, calleeWallet.getAddress(), oneToken);
        tokenScore.ensureTransfer(result, ownerWallet.getAddress(), calleeWallet.getAddress(), oneToken, null);
        LOG.infoExiting();

        // 3.1 transfer #2
        LOG.infoEntering("transfer", "#2");
        BigInteger two = oneToken.add(oneToken);
        byte[] data = "Hello".getBytes();
        result = tokenScore.transfer(ownerWallet, calleeWallet.getAddress(), two, data);
        assertSuccess(result);
        tokenScore.ensureTransfer(result, ownerWallet.getAddress(), calleeWallet.getAddress(), two, data);
        LOG.infoExiting();

        // 4. check balance of callee
        LOG.infoEntering("balanceOf", "callee");
        BigInteger expected = oneToken.add(two);
        bal = tokenScore.balanceOf(calleeWallet.getAddress());
        LOG.info("expected (" + expected + "), got (" + bal + ")");
        assertEquals(expected, bal);
        LOG.infoExiting();

        // 5. check balance of owner
        LOG.infoEntering("balanceOf", "owner");
        expected = totalSupply.subtract(expected);
        bal = tokenScore.balanceOf(ownerWallet.getAddress());
        LOG.info("expected (" + expected + "), got (" + bal + ")");
        assertEquals(expected, bal);
        LOG.infoExiting();
    }

    @Test
    public void testIRC3Token() throws Exception {
        // 1. deploy
        IRC3TokenScore tokenScore = IRC3TokenScore.mustDeploy(
                txHandler, ownerWallet, new Class<?>[]{
                        IRC3BasicToken.class, IRC3Basic.class, IRC3.class,
                        EnumerableIntMap.class, EnumerableSet.class, IntSet.class});

        // 2. initial check
        LOG.infoEntering("initial check");
        assertEquals(BigInteger.ZERO, tokenScore.balanceOf(ownerWallet.getAddress()));
        assertEquals(BigInteger.ZERO, tokenScore.balanceOf(caller.getAddress()));
        assertEquals(BigInteger.ZERO, tokenScore.totalSupply());
        LOG.infoExiting();

        // 3. mint some tokens
        LOG.infoEntering("mint some tokens");
        BigInteger[] tokenId = new BigInteger[] {
                new BigInteger(getRandomBytes(8)),
                new BigInteger(getRandomBytes(8)),
                new BigInteger(getRandomBytes(8)),
                new BigInteger(getRandomBytes(8)),
        };
        Bytes[] ids = new Bytes[tokenId.length];
        for (int i = 0; i < ids.length; i++) {
            ids[i] = tokenScore.mint(ownerWallet, tokenId[i]);
        }
        for (Bytes id : ids) {
            assertSuccess(txHandler.getResult(id));
        }
        assertEquals(BigInteger.valueOf(tokenId.length), tokenScore.balanceOf(ownerWallet.getAddress()));
        assertEquals(BigInteger.valueOf(tokenId.length), tokenScore.totalSupply());
        showTokenStatus(tokenScore);
        LOG.infoExiting();

        // 4. transfer and check
        LOG.infoEntering("transfer and check");
        BigInteger token = tokenId[0];
        assertEquals(ownerWallet.getAddress(), tokenScore.ownerOf(token));
        ids[0] = tokenScore.transfer(ownerWallet, caller.getAddress(), token);
        assertSuccess(txHandler.getResult(ids[0]));
        assertEquals(caller.getAddress(), tokenScore.ownerOf(token));
        assertEquals(BigInteger.ONE, tokenScore.balanceOf(caller.getAddress()));
        assertEquals(BigInteger.valueOf(tokenId.length-1), tokenScore.balanceOf(ownerWallet.getAddress()));
        assertEquals(BigInteger.valueOf(tokenId.length), tokenScore.totalSupply());
        assertEquals(token, tokenScore.tokenOfOwnerByIndex(caller.getAddress(), 0));
        assertEquals(tokenId[tokenId.length-1], tokenScore.tokenOfOwnerByIndex(ownerWallet.getAddress(), 0));
        showTokenStatus(tokenScore);
        LOG.infoExiting();

        // 5. approve and check
        LOG.infoEntering("approve and check");
        token = tokenId[1];
        assertEquals(ZERO_ADDRESS, tokenScore.getApproved(token));
        ids[1] = tokenScore.approve(ownerWallet, caller.getAddress(), token);
        assertSuccess(txHandler.getResult(ids[1]));
        assertEquals(caller.getAddress(), tokenScore.getApproved(token));
        showTokenStatus(tokenScore);

        assertEquals(ownerWallet.getAddress(), tokenScore.ownerOf(token));
        ids[2] = tokenScore.transferFrom(caller, ownerWallet.getAddress(), caller.getAddress(), token);
        assertSuccess(txHandler.getResult(ids[2]));
        assertEquals(ZERO_ADDRESS, tokenScore.getApproved(token));
        assertEquals(caller.getAddress(), tokenScore.ownerOf(token));
        assertEquals(BigInteger.TWO, tokenScore.balanceOf(caller.getAddress()));
        assertEquals(BigInteger.valueOf(tokenId.length-2), tokenScore.balanceOf(ownerWallet.getAddress()));
        assertEquals(BigInteger.valueOf(tokenId.length), tokenScore.totalSupply());
        assertEquals(token, tokenScore.tokenOfOwnerByIndex(caller.getAddress(), 1));
        assertEquals(tokenId[tokenId.length-2], tokenScore.tokenOfOwnerByIndex(ownerWallet.getAddress(), 1));
        showTokenStatus(tokenScore);
        LOG.infoExiting();

        // 6. burn and check
        LOG.infoEntering("burn and check");
        var balance = tokenScore.balanceOf(ownerWallet.getAddress());
        token = tokenScore.tokenOfOwnerByIndex(ownerWallet.getAddress(), 0);
        ids[0] = tokenScore.burn(ownerWallet, token);
        assertSuccess(txHandler.getResult(ids[0]));
        assertEquals(balance.subtract(BigInteger.ONE), tokenScore.balanceOf(ownerWallet.getAddress()));
        assertEquals(BigInteger.valueOf(tokenId.length-1), tokenScore.totalSupply());
        showTokenStatus(tokenScore);
        LOG.infoExiting();

        // 7. negative tests
        LOG.infoEntering("negative tests");
        final var nonExistToken = token; // burned token
        assertThrows(RpcError.class, () -> tokenScore.ownerOf(nonExistToken));
        assertFailure(txHandler.getResult(
                tokenScore.transferFrom(caller, ownerWallet.getAddress(), caller.getAddress(), tokenId[2])));
        // out of the supported range
        byte[] randBytes = getRandomBytes(66);
        if (randBytes[1] == 0) {
            randBytes[1] = (byte)0xFF;
        }
        final var bigLengthToken = new BigInteger(randBytes);
        try {
            tokenScore.ownerOf(bigLengthToken);
        } catch (RpcError e) {
            final int ErrInvalidParameter = -30006;
            assertEquals(ErrInvalidParameter, e.getCode());
            LOG.info("Expected RpcError: code=" + e.getCode() + ", msg=" + e.getMessage());
        }
        LOG.infoExiting();
    }

    private void showTokenStatus(IRC3TokenScore tokenScore) throws Exception {
        var totalSupply = tokenScore.totalSupply();
        System.out.println(">>> totalSupply = " + totalSupply);
        for (int i = 0; i < totalSupply.intValue(); i++) {
            var token = tokenScore.tokenByIndex(i);
            var owner = tokenScore.ownerOf(token);
            var approved = tokenScore.getApproved(token);
            System.out.printf("   [%s](%s)<%s>\n", token, owner,
                    approved.equals(ZERO_ADDRESS) ? "null" : approved);
        }
        var ownerBalance = tokenScore.balanceOf(ownerWallet.getAddress());
        System.out.println("   == balanceOf owner: " + ownerBalance);
        for (int i = 0; i < ownerBalance.intValue(); i++) {
            var token = tokenScore.tokenOfOwnerByIndex(ownerWallet.getAddress(), i);
            System.out.printf("     -- %d: [%s]\n", i, token);
        }
        var callerBalance = tokenScore.balanceOf(caller.getAddress());
        System.out.println("   == balanceOf caller: " + callerBalance);
        for (int i = 0; i < callerBalance.intValue(); i++) {
            var token = tokenScore.tokenOfOwnerByIndex(caller.getAddress(), i);
            System.out.printf("     -- %d: [%s]\n", i, token);
        }
    }
}
