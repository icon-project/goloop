/*
 * Copyright 2020 ICON Foundation
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

package foundation.icon.test.score;

import foundation.icon.icx.Wallet;
import foundation.icon.icx.data.Address;
import foundation.icon.icx.data.Bytes;
import foundation.icon.icx.transport.jsonrpc.RpcObject;
import foundation.icon.icx.transport.jsonrpc.RpcValue;
import foundation.icon.test.common.ResultTimeoutException;
import foundation.icon.test.common.TransactionFailureException;
import foundation.icon.test.common.TransactionHandler;

import java.io.IOException;
import java.math.BigInteger;

import static foundation.icon.test.common.Env.LOG;

public class IRC3TokenScore extends Score {
    private static final BigInteger MINT_STEP = BigInteger.valueOf(400000);
    private static final BigInteger TRANSFER_STEP = BigInteger.valueOf(500000);

    public IRC3TokenScore(Score other) {
        super(other);
    }

    public static IRC3TokenScore mustDeploy(TransactionHandler txHandler, Wallet owner, Class<?>[] classes)
            throws ResultTimeoutException, TransactionFailureException, IOException {
        LOG.infoEntering("deploy", classes[0].getName());
        RpcObject params = new RpcObject.Builder()
                .put("_name", new RpcValue("MyNFT"))
                .put("_symbol", new RpcValue("NFT"))
                .build();
        Score score = txHandler.deploy(owner, classes, params);
        LOG.info("scoreAddr = " + score.getAddress());
        LOG.infoExiting();
        return new IRC3TokenScore(score);
    }

    public Address ownerOf(BigInteger tokenId) throws IOException {
        RpcObject params = new RpcObject.Builder()
                .put("_tokenId", new RpcValue(tokenId))
                .build();
        return call("ownerOf", params).asAddress();
    }

    public Address getApproved(BigInteger tokenId) throws IOException {
        RpcObject params = new RpcObject.Builder()
                .put("_tokenId", new RpcValue(tokenId))
                .build();
        return call("getApproved", params).asAddress();
    }

    public BigInteger balanceOf(Address owner) throws IOException {
        RpcObject params = new RpcObject.Builder()
                .put("_owner", new RpcValue(owner))
                .build();
        return call("balanceOf", params).asInteger();
    }

    public BigInteger totalSupply() throws IOException {
        return call("totalSupply", null).asInteger();
    }

    public BigInteger tokenByIndex(int index) throws IOException {
        RpcObject params = new RpcObject.Builder()
                .put("_index", new RpcValue(BigInteger.valueOf(index)))
                .build();
        return call("tokenByIndex", params).asInteger();
    }

    public BigInteger tokenOfOwnerByIndex(Address owner, int index) throws IOException {
        RpcObject params = new RpcObject.Builder()
                .put("_owner", new RpcValue(owner))
                .put("_index", new RpcValue(BigInteger.valueOf(index)))
                .build();
        return call("tokenOfOwnerByIndex", params).asInteger();
    }

    public Bytes mint(Wallet wallet, BigInteger tokenId) throws IOException {
        RpcObject params = new RpcObject.Builder()
                .put("_tokenId", new RpcValue(tokenId))
                .build();
        return invoke(wallet, "mint", params, null, MINT_STEP);
    }

    public Bytes burn(Wallet wallet, BigInteger tokenId) throws IOException {
        RpcObject params = new RpcObject.Builder()
                .put("_tokenId", new RpcValue(tokenId))
                .build();
        return invoke(wallet, "burn", params, null, MINT_STEP);
    }

    public Bytes approve(Wallet wallet, Address to, BigInteger tokenId) throws IOException {
        RpcObject params = new RpcObject.Builder()
                .put("_to", new RpcValue(to))
                .put("_tokenId", new RpcValue(tokenId))
                .build();
        return invoke(wallet, "approve", params, null, MINT_STEP);
    }

    public Bytes transfer(Wallet wallet, Address to, BigInteger tokenId) throws IOException {
        RpcObject params = new RpcObject.Builder()
                .put("_to", new RpcValue(to))
                .put("_tokenId", new RpcValue(tokenId))
                .build();
        return invoke(wallet, "transfer", params, null, TRANSFER_STEP);
    }

    public Bytes transferFrom(Wallet wallet, Address from, Address to, BigInteger tokenId) throws IOException {
        RpcObject params = new RpcObject.Builder()
                .put("_from", new RpcValue(from))
                .put("_to", new RpcValue(to))
                .put("_tokenId", new RpcValue(tokenId))
                .build();
        return invoke(wallet, "transferFrom", params, null, TRANSFER_STEP);
    }
}
