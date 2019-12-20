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

package foundation.icon.test.score;

import foundation.icon.icx.*;
import foundation.icon.icx.data.Address;
import foundation.icon.icx.data.Bytes;
import foundation.icon.icx.data.IconAmount;
import foundation.icon.icx.data.TransactionResult;
import foundation.icon.icx.transport.jsonrpc.RpcObject;
import foundation.icon.icx.transport.jsonrpc.RpcValue;
import foundation.icon.test.common.*;

import java.io.IOException;
import java.math.BigDecimal;
import java.math.BigInteger;

import static foundation.icon.test.common.Env.LOG;

public class SampleTokenScore extends Score {
    private static final String PATH = Constants.SCORE_SAMPLETOKEN_PATH;

    public static SampleTokenScore mustDeploy(IconService service, Env.Chain chain, Wallet wallet,
                                              String name, String symbol, int decimals, BigInteger initialSupply)
            throws ResultTimeoutException, TransactionFailureException, IOException
    {
        RpcObject params = new RpcObject.Builder()
                .put("_name", new RpcValue(name))
                .put("_symbol", new RpcValue(symbol))
                .put("_decimals", new RpcValue(BigInteger.valueOf(decimals)))
                .put("_initialSupply", new RpcValue(initialSupply))
                .build();
        return new SampleTokenScore(
                service,
                chain,
                Score.install(service, chain, wallet, PATH, params)
        );
    }

    public SampleTokenScore(IconService iconService, Env.Chain chain, Address scoreAddress) {
        super(iconService, chain, scoreAddress);

        //TODO: check if this is really a token SCORE that conforms to IRC2
    }

    public BigInteger balanceOf(Address owner) throws IOException {
        RpcObject params = new RpcObject.Builder()
                .put("_owner", new RpcValue(owner))
                .build();
        return call("balanceOf", params).asInteger();
    }

    public void ensureTokenBalance(KeyWallet wallet, long value) throws ResultTimeoutException, IOException {
        long limitTime = System.currentTimeMillis() + Constants.DEFAULT_WAITING_TIME;
        while (true) {
            BigInteger balance = balanceOf(wallet.getAddress());
            String msg = "Token balance of " + wallet.getAddress() + ": " + balance;
            if (balance.equals(BigInteger.valueOf(0))) {
                try {
                    if (limitTime < System.currentTimeMillis()) {
                        throw new ResultTimeoutException();
                    }
                    // wait until block confirmation
                    LOG.info(msg + "; Retry in 1 sec.");
                    Thread.sleep(1000);
                } catch (InterruptedException e) {
                    e.printStackTrace();
                }
            } else if (balance.equals(BigInteger.valueOf(value).multiply(BigDecimal.TEN.pow(18).toBigInteger()))) {
                LOG.info(msg);
                break;
            } else {
                throw new IOException("Token balance mismatch!");
            }
        }
    }

    public Bytes transfer(Wallet fromWallet, Address toAddress, BigInteger value) throws IOException {
        RpcObject params = new RpcObject.Builder()
                .put("_to", new RpcValue(toAddress))
                .put("_value", new RpcValue(IconAmount.of(value, 18).toLoop()))
                .build();
        return this.invoke(fromWallet, "transfer", params, null, Constants.DEFAULT_STEPS,
                Utils.getMicroTime(), BigInteger.ONE);
    }

    public void ensureFundTransfer(TransactionResult result, Address scoreAddress,
                                   Address backer, BigInteger amount) throws IOException {
        TransactionResult.EventLog event = Utils.findEventLogWithFuncSig(result, scoreAddress, "FundTransfer(Address,int,bool)");
        if (event != null) {
            Address _backer = event.getIndexed().get(1).asAddress();
            BigInteger _amount = event.getIndexed().get(2).asInteger();
            Boolean isContribution = event.getIndexed().get(3).asBoolean();
            if (backer.equals(_backer) && amount.equals(_amount) && !isContribution) {
                return; // ensured
            }
        }
        throw new IOException("ensureFundTransfer failed.");
    }
}
