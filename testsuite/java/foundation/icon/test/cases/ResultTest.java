package foundation.icon.test.cases;

import foundation.icon.icx.IconService;
import foundation.icon.icx.KeyWallet;
import foundation.icon.icx.data.Address;
import foundation.icon.icx.data.Bytes;
import foundation.icon.icx.data.TransactionResult;
import foundation.icon.icx.transport.http.HttpProvider;
import foundation.icon.icx.transport.jsonrpc.RpcError;
import foundation.icon.icx.transport.jsonrpc.RpcItem;
import foundation.icon.icx.transport.jsonrpc.RpcObject;
import foundation.icon.icx.transport.jsonrpc.RpcValue;
import foundation.icon.test.common.Constants;
import foundation.icon.test.common.Env;
import foundation.icon.test.score.Score;
import org.junit.jupiter.api.BeforeAll;
import org.junit.jupiter.api.Tag;
import org.junit.jupiter.api.Test;

import java.math.BigInteger;

import static foundation.icon.test.common.Env.LOG;
import static org.junit.jupiter.api.Assertions.*;

@Tag(Constants.TAG_NORMAL)
public class ResultTest {
    private static Env.Chain chain;
    private static IconService iconService;

    private static KeyWallet ownerWallet;

    private static Score score1, score2;
    private static Score chainSCORE;

    @BeforeAll
    public static void setUp() throws Exception {
        Env.Node node = Env.nodes[0];
        Env.Channel channel = node.channels[0];
        chain = channel.chain;
        iconService = new IconService(new HttpProvider(channel.getAPIUrl(Env.testApiVer)));
        ownerWallet = KeyWallet.create();

        score1 = new Score(iconService, chain,
                Score.install(iconService, chain, ownerWallet,
                        Constants.SCORE_ROOT + "result_gen",
                        null));
        score2 = new Score(iconService, chain,
                Score.install(iconService, chain, ownerWallet,
                        Constants.SCORE_ROOT + "result_gen",
                        null));
        chainSCORE = new Score(iconService, chain,
                new Address("cx0000000000000000000000000000000000000000"));
    }

    final static int CODE_REVERTED = 32;
    final static int CODE_LIMIT_REV5 = 99;
    final static int CODE_LIMIT_REV6 = 999;

    @Test
    void checkFailureCodeForRevert() throws Exception {
        LOG.infoEntering("checkFailureCodeForRevert");
        BigInteger[] cases = {
                BigInteger.ZERO,
                BigInteger.valueOf(CODE_LIMIT_REV5 - CODE_REVERTED + 1),
                BigInteger.valueOf(CODE_LIMIT_REV6 - CODE_REVERTED + 1),
        };
        BigInteger[] expect;
        RpcItem rev = chainSCORE.call(null, "getRevision", null);
        LOG.info("Revision: " + rev.asString());
        if (rev.asInteger().intValue() < 6) {
            expect = new BigInteger[]{
                    BigInteger.valueOf(CODE_REVERTED),
                    BigInteger.valueOf(CODE_LIMIT_REV5),
                    BigInteger.valueOf(CODE_LIMIT_REV5),
            };
        } else {
            expect = new BigInteger[]{
                    BigInteger.valueOf(CODE_REVERTED),
                    BigInteger.valueOf(CODE_LIMIT_REV5 + 1),
                    BigInteger.valueOf(CODE_LIMIT_REV6),
            };
        }

        Bytes[] txs = new Bytes[cases.length];
        Bytes[] icTxs = new Bytes[cases.length];
        Bytes[] iccTxs = new Bytes[cases.length];
        for (int i = 0; i < cases.length; i++) {
            LOG.info("send tx normal case" + String.valueOf(i));
            RpcObject params = new RpcObject.Builder()
                    .put("index", new RpcValue(cases[i]))
                    .build();
            txs[i] = score1.invoke(
                    ownerWallet, "callRevertWithIndex", params,
                    0, Constants.DEFAULT_STEP_LIMIT);

            LOG.info("query case" + String.valueOf(i));
            try {
                RpcItem qr = score1.call(null, "queryRevertWithIndex", params);
                fail();
            } catch (RpcError e) {
                assertEquals(-30000 - expect[i].intValue(), e.getCode());
            }

            LOG.info("send tx inter-call case" + String.valueOf(i));
            params = new RpcObject.Builder()
                    .put("addr", new RpcValue(score2.getAddress()))
                    .put("index", new RpcValue(cases[i]))
                    .build();
            icTxs[i] = score1.invoke(
                    ownerWallet, "interCallRevertWithIndex", params,
                    0, Constants.DEFAULT_STEP_LIMIT);

            LOG.info("send tx inter-call catch case" + i);
            iccTxs[i] = score1.invoke(
                    ownerWallet, "interCallCatchRevertWithIndex", params,
                    0, Constants.DEFAULT_STEP_LIMIT);
        }

        for (int i = 0; i < cases.length; i++) {
            LOG.info("check result for normal case" + i);
            TransactionResult result = score1.waitResult(txs[i]);
            assertEquals(result.getStatus(), Constants.STATUS_FAIL);
            assertEquals(expect[i], result.getFailure().getCode());

            LOG.info("check result for inter-call case" + i);
            result = score1.waitResult(icTxs[i]);
            assertEquals(result.getStatus(), Constants.STATUS_FAIL);
            assertEquals(expect[i], result.getFailure().getCode());

            LOG.info("check result for inter-call catch case" + i);
            result = score1.waitResult(iccTxs[i]);
            assertEquals(result.getStatus(), Constants.STATUS_SUCCESS);
            boolean checked = false;
            for (TransactionResult.EventLog el : result.getEventLogs()) {
                if (el.getIndexed().get(0).asString().equals("RevertCatch(int)")) {
                    assertEquals(expect[i], el.getData().get(0).asInteger());
                    checked = true;
                }
            }
            assertTrue(checked);
        }

        LOG.infoExiting();
    }

    @Test
    void checkExternalReturnValue() throws Exception {
        String[] values = {
                "HelloWorld", "한글", ""
        };

        Bytes[] txs = new Bytes[values.length];
        for (int i = 0; i < values.length; i++) {
            RpcObject params = new RpcObject.Builder()
                    .put("addr", new RpcValue(score2.getAddress()))
                    .put("value", new RpcValue(values[i]))
                    .build();
            txs[i] = score1.invoke(ownerWallet, "interCallReturnStr", params,
                    0, Constants.DEFAULT_STEP_LIMIT);
        }

        for (int i = 0; i < values.length; i++) {
            TransactionResult result = score1.waitResult(txs[i]);
            assertEquals(Constants.STATUS_SUCCESS, result.getStatus());
            int checked = 0;
            for (TransactionResult.EventLog el : result.getEventLogs()) {
                if (el.getIndexed().get(0).asString().equals("ReturnedStr(str)")) {
                    assertEquals(values[i], el.getData().get(0).asString());
                    checked += 1;
                }
            }
            assertEquals(1, checked);
        }
    }
}
