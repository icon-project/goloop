package foundation.icon.test.cases;

import foundation.icon.icx.IconService;
import foundation.icon.icx.KeyWallet;
import foundation.icon.icx.data.Address;
import foundation.icon.icx.transport.http.HttpProvider;
import foundation.icon.test.common.Constants;
import foundation.icon.test.common.Env;
import foundation.icon.test.common.ResultTimeoutException;
import foundation.icon.test.common.TBCTestScenario;
import test.TBCProtocol;
import foundation.icon.test.common.TestBase;
import foundation.icon.test.common.TransactionHandler;
import foundation.icon.test.score.TBCInterpreterScore;
import org.junit.jupiter.api.BeforeAll;
import org.junit.jupiter.api.Tag;
import org.junit.jupiter.api.Test;

import java.io.IOException;
import java.util.List;

import static org.junit.jupiter.api.Assertions.assertEquals;

import static foundation.icon.test.common.Env.LOG;

public class TBCTest extends TestBase {
    private static final int S = TBCProtocol.VAR_TYPE_STATIC;
    private static final int I = TBCProtocol.VAR_TYPE_INSTANCE;
    private static final int L = TBCProtocol.VAR_TYPE_LOCAL;

    private static final char[] HEX_ARRAY = "0123456789abcdef".toCharArray();

    private static TransactionHandler txHandler;
    private static KeyWallet ownerWallet;

    private TBCInterpreterScore s1;
    private TBCInterpreterScore s2;

    @BeforeAll
    static void setup() throws Exception {
        Env.Node node = Env.nodes[0];
        Env.Channel channel = node.channels[0];
        Env.Chain chain = channel.chain;
        IconService iconService = new IconService(new HttpProvider(channel.getAPIUrl(Env.testApiVer)));
        txHandler = new TransactionHandler(iconService, chain);
        ownerWallet = KeyWallet.create();
    }

    private static String toHex(byte[] bytes) {
        char[] hexChars = new char[bytes.length * 2];
        for (int j = 0; j < bytes.length; j++) {
            int v = bytes[j] & 0xFF;
            hexChars[j * 2] = HEX_ARRAY[v >>> 4];
            hexChars[j * 2 + 1] = HEX_ARRAY[v & 0x0F];
        }
        return new String(hexChars);
    }

    private void subcase(String name, TBCTestScenario scenario)
            throws IOException, ResultTimeoutException {
        var totalExp = scenario.getExpectCount();
        var tr = s1.runAndLogEvent(ownerWallet, scenario.compile());
        var out = tr.getEventLogs().get(0).getIndexed().get(1).asString();
        var list = List.of(out.split("\n"));
        var okObs = list
                .stream()
                .filter(s -> s.startsWith("EXPECT"))
                .filter(s -> s.contains("[OK]"))
                .count();
        var totalObs = list
                .stream()
                .filter(s -> s.startsWith("EXPECT"))
                .count();
        LOG.info(String.format("%s : %d/%d", name, okObs, totalObs));
        if (okObs != totalExp) {
            LOG.info("debug log:\n" + out);
        }
        assertEquals(totalObs, totalExp);
        assertEquals(totalObs, okObs);
    }

    TBCTestScenario newValueScenario(int type, Address a1, Address a2) {
        return new TBCTestScenario()
                .set(type, 0, "0")
                .call(a2)
                    .call(a1)
                        .expect(type, 0, "0")
                        .set(type, 0, "1")
                    .revert()
                    .call(a1)
                        .expect(type, 0, "0")
                        .set(type, 0, "2")
                    .ret()
                    .call(a1)
                        .expect(type, 0, "2")
                        .set(type, 0, "3")
                    .ret()
                .revert()
                .expect(type, 0, "0")
                .call(a2)
                    .call(a1)
                        .expect(type, 0, "0")
                        .set(type, 0, "4")
                    .revert()
                    .call(a1)
                        .expect(type, 0, "0")
                        .set(type, 0, "5")
                    .ret()
                    .call(a1)
                        .expect(type, 0, "5")
                        .set(type, 0, "6")
                    .ret()
                .ret()
                .expect(type, 0, "6");
    }

    TBCTestScenario newRefScenario(int type, Address a1, Address a2) {
        return new TBCTestScenario()
                .set(S, TBCProtocol.MAX_VAR-1, "s-1")
                .set(S, TBCProtocol.MAX_VAR-2, "s-2")
                .set(S, TBCProtocol.MAX_VAR-3, "s-3")
                .set(S, TBCProtocol.MAX_VAR-4, "s-4")
                .setRef(type, 0, S, TBCProtocol.MAX_VAR-1)
                .setRef(type, 1, S, TBCProtocol.MAX_VAR-2)
                .setRef(type, 2, S, TBCProtocol.MAX_VAR-3)
                .call(a2)
                    .call(a1)
                        .expectRef(type, 0, S, TBCProtocol.MAX_VAR-1)
                        .expectRef(type, 1, S, TBCProtocol.MAX_VAR-2)
                        .expectRef(type, 2, S, TBCProtocol.MAX_VAR-3)
                        .set(S, TBCProtocol.MAX_VAR-1, "s-1")
                        .setRef(type, 1, S, TBCProtocol.MAX_VAR-3)
                    .revert()
                    .call(a1)
                        .expectRef(type, 0, S, TBCProtocol.MAX_VAR-1)
                        .expectRef(type, 1, S, TBCProtocol.MAX_VAR-2)
                        .expectRef(type, 2, S, TBCProtocol.MAX_VAR-3)
                        .set(S, TBCProtocol.MAX_VAR-1, "s-1")
                        .setRef(type, 1, S, TBCProtocol.MAX_VAR-3)
                    .ret()
                    .call(a1)
                        .expectRefNE(type, 0, S, TBCProtocol.MAX_VAR-1)
                        .expectRefNE(type, 1, S, TBCProtocol.MAX_VAR-2)
                        .expectRef(type, 1, S, TBCProtocol.MAX_VAR-3)
                        .expectRef(type, 2, S, TBCProtocol.MAX_VAR-3)
                        .setRef(type, 1, S, TBCProtocol.MAX_VAR-4)
                    .ret()
                .revert()
                .expectRef(type, 0, S, TBCProtocol.MAX_VAR-1)
                .expectRef(type, 1, S, TBCProtocol.MAX_VAR-2)
                .expectRef(type, 2, S, TBCProtocol.MAX_VAR-3)
                .call(a2)
                    .call(a1)
                        .expectRef(type, 0, S, TBCProtocol.MAX_VAR-1)
                        .expectRef(type, 1, S, TBCProtocol.MAX_VAR-2)
                        .expectRef(type, 2, S, TBCProtocol.MAX_VAR-3)
                        .set(S, TBCProtocol.MAX_VAR-1, "s-1")
                        .setRef(type, 1, S, TBCProtocol.MAX_VAR-3)
                    .revert()
                    .call(a1)
                        .expectRef(type, 0, S, TBCProtocol.MAX_VAR-1)
                        .expectRef(type, 1, S, TBCProtocol.MAX_VAR-2)
                        .expectRef(type, 2, S, TBCProtocol.MAX_VAR-3)
                        .set(S, TBCProtocol.MAX_VAR-1, "s-1")
                        .setRef(type, 1, S, TBCProtocol.MAX_VAR-3)
                    .ret()
                    .call(a1)
                        .expectRefNE(type, 0, S, TBCProtocol.MAX_VAR-1)
                        .expectRefNE(type, 1, S, TBCProtocol.MAX_VAR-2)
                        .expectRef(type, 1, S, TBCProtocol.MAX_VAR-3)
                        .expectRef(type, 2, S, TBCProtocol.MAX_VAR-3)
                        .setRef(type, 1, S, TBCProtocol.MAX_VAR-4)
                    .ret()
                .ret()
                .expectRefNE(type, 0, S, TBCProtocol.MAX_VAR-1)
                .expectRefNE(type, 1, S, TBCProtocol.MAX_VAR-2)
                .expectRef(type, 1, S, TBCProtocol.MAX_VAR-4)
                .expectRef(type, 2, S, TBCProtocol.MAX_VAR-3);
    }

    TBCTestScenario newLocalRefScenario(Address a1, Address a2) {
        return new TBCTestScenario()
                .set(S, TBCProtocol.MAX_VAR-1, "s-1")
                .set(S, TBCProtocol.MAX_VAR-2, "s-2")
                .set(S, TBCProtocol.MAX_VAR-3, "s-3")
                .set(S, TBCProtocol.MAX_VAR-4, "s-4")
                .setRef(L, 0, S, TBCProtocol.MAX_VAR-1)
                .setRef(L, 1, S, TBCProtocol.MAX_VAR-2)
                .setRef(L, 2, S, TBCProtocol.MAX_VAR-3)
                .call(a2)
                    .call(a1)
                        .set(S, TBCProtocol.MAX_VAR-1, "s-1")
                        .setRef(L, 1, S, TBCProtocol.MAX_VAR-4)
                    .revert()
                .revert()
                .expectRef(L, 0, S, TBCProtocol.MAX_VAR-1)
                .expectRef(L, 1, S, TBCProtocol.MAX_VAR-2)
                .expectRef(L, 2, S, TBCProtocol.MAX_VAR-3)
                .call(a2)
                    .call(a1)
                        .set(S, TBCProtocol.MAX_VAR-1, "s-1")
                        .setRef(L, 1, S, TBCProtocol.MAX_VAR-4)
                    .ret()
                .revert()
                .expectRef(L, 0, S, TBCProtocol.MAX_VAR-1)
                .expectRef(L, 1, S, TBCProtocol.MAX_VAR-2)
                .expectRef(L, 2, S, TBCProtocol.MAX_VAR-3)
                .call(a2)
                    .call(a1)
                        .set(S, TBCProtocol.MAX_VAR-1, "s-1")
                        .setRef(L, 1, S, TBCProtocol.MAX_VAR-4)
                    .revert()
                .ret()
                .expectRef(L, 0, S, TBCProtocol.MAX_VAR-1)
                .expectRef(L, 1, S, TBCProtocol.MAX_VAR-2)
                .expectRef(L, 2, S, TBCProtocol.MAX_VAR-3)
                .call(a2)
                    .call(a1)
                        .set(S, TBCProtocol.MAX_VAR-1, "s-1")
                        .setRef(L, 1, S, TBCProtocol.MAX_VAR-4)
                    .ret()
                .ret()
                .expectRefNE(L, 0, S, TBCProtocol.MAX_VAR-1)
                .expectRefNE(L, 1, S, TBCProtocol.MAX_VAR-2)
                .expectRef(L, 1, S, TBCProtocol.MAX_VAR-4)
                .expectRef(L, 2, S, TBCProtocol.MAX_VAR-3);
    }

    @Tag(Constants.TAG_INTER_SCORE)
    @Test
    void testSimpleScenario() throws Exception {
        s1 = TBCInterpreterScore.mustDeploy(txHandler, ownerWallet,
                "s1", Constants.CONTENT_TYPE_JAVA);
        s2 = TBCInterpreterScore.mustDeploy(txHandler, ownerWallet,
                "s2", Constants.CONTENT_TYPE_PYTHON);
        LOG.infoEntering("run scenario");

        var a1 = s1.getAddress();
        var a2 = s2.getAddress();

        subcase("Static value direct", newValueScenario(S, a1, a1));
        subcase("Instance value direct", newValueScenario(I, a1, a1));
        subcase("Static value indirect", newValueScenario(S, a1, a2));
        subcase("Instance value indirect", newValueScenario(I, a1, a2));

        subcase("Static ref direct", newRefScenario(S, a1, a1));
        subcase("Instance ref direct", newRefScenario(I, a1, a1));
        subcase("Static ref indirect", newRefScenario(S, a1, a2));
        subcase("Instance ref indirect", newRefScenario(I, a1, a2));

        subcase("Local ref direct", newRefScenario(S, a1, a1));
        subcase("Local ref indirect", newRefScenario(S, a1, a2));

        LOG.infoExiting();
    }
}