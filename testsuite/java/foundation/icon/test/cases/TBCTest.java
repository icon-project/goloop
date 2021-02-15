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

package foundation.icon.test.cases;

import foundation.icon.ee.test.TBCProtocol;
import foundation.icon.ee.test.TBCTestScenario;
import foundation.icon.ee.test.TBCTestScenarios;
import foundation.icon.icx.IconService;
import foundation.icon.icx.KeyWallet;
import foundation.icon.icx.data.Address;
import foundation.icon.icx.transport.http.HttpProvider;
import foundation.icon.test.common.Constants;
import foundation.icon.test.common.Env;
import foundation.icon.test.common.ResultTimeoutException;
import foundation.icon.test.common.TestBase;
import foundation.icon.test.common.TransactionHandler;
import foundation.icon.test.score.TBCInterpreterScore;
import org.junit.jupiter.api.BeforeAll;
import org.junit.jupiter.api.Tag;
import org.junit.jupiter.api.Test;

import java.io.IOException;
import java.math.BigInteger;

import static foundation.icon.test.common.Env.LOG;
import static org.junit.jupiter.api.Assertions.assertEquals;

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
        var okObs = tr.getEventLogs().get(0).getIndexed().get(1).asInteger();
        LOG.info(String.format("%s : %d/%d", name, okObs, totalExp));
        assertEquals(BigInteger.valueOf(totalExp), okObs);
    }

    byte[] toByteArray(Address a) {
        var body = a.getBody();
        var out = new byte[body.length + 1];
        out[0] = (byte) (a.getPrefix().ordinal() & 0xff);
        System.arraycopy(body, 0, out, 1, body.length);
        return out;
    }

    @Tag(Constants.TAG_JAVA_GOV)
    @Test
    void testSimpleScenario() throws Exception {
        s1 = TBCInterpreterScore.mustDeploy(txHandler, ownerWallet,
                "s1", Constants.CONTENT_TYPE_JAVA);
        s2 = TBCInterpreterScore.mustDeploy(txHandler, ownerWallet,
                "s2", Constants.CONTENT_TYPE_PYTHON);
        LOG.infoEntering("run scenario");

        var a1 = toByteArray(s1.getAddress());
        var a2 = toByteArray(s2.getAddress());

        subcase("Static value direct", TBCTestScenarios.newValueScenario(S, a1, a1));
        subcase("Instance value direct", TBCTestScenarios.newValueScenario(I, a1, a1));
        subcase("Static value indirect", TBCTestScenarios.newValueScenario(S, a1, a2));
        subcase("Instance value indirect", TBCTestScenarios.newValueScenario(I, a1, a2));

        subcase("Static ref direct", TBCTestScenarios.newRefScenario(S, a1, a1));
        subcase("Instance ref direct", TBCTestScenarios.newRefScenario(I, a1, a1));
        subcase("Static ref indirect", TBCTestScenarios.newRefScenario(S, a1, a2));
        subcase("Instance ref indirect", TBCTestScenarios.newRefScenario(I, a1, a2));

        subcase("Local/Instance ref direct", TBCTestScenarios.newLocalRefScenario(I, a1, a1));
        subcase("Local/Instance ref indirect", TBCTestScenarios.newLocalRefScenario(I, a1, a2));
        subcase("Local/Static ref direct", TBCTestScenarios.newLocalRefScenario(S, a1, a1));
        subcase("Local/Static ref indirect", TBCTestScenarios.newLocalRefScenario(S, a1, a2));

        LOG.infoExiting();
    }
}
