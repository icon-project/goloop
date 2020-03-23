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
import foundation.icon.icx.data.TransactionResult;
import foundation.icon.icx.transport.http.HttpProvider;
import foundation.icon.test.common.Constants;
import foundation.icon.test.common.Env;
import foundation.icon.test.common.TestBase;
import foundation.icon.test.common.TransactionHandler;
import foundation.icon.test.score.StepCounterScore;
import org.junit.jupiter.api.BeforeAll;
import org.junit.jupiter.api.Tag;
import org.junit.jupiter.api.Test;

import java.math.BigInteger;

import static foundation.icon.test.common.Env.LOG;
import static org.junit.jupiter.api.Assertions.assertEquals;

@Tag(Constants.TAG_PY_SCORE)
public class RevertTest extends TestBase {
    private static TransactionHandler txHandler;

    @BeforeAll
    static void setup() {
        Env.Node node = Env.nodes[0];
        Env.Channel channel = node.channels[0];
        Env.Chain chain = channel.chain;
        IconService iconService = new IconService(new HttpProvider(channel.getAPIUrl(Env.testApiVer)));
        txHandler = new TransactionHandler(iconService, chain);
    }

    @Test
    public void runTest() throws Exception {
        KeyWallet ownerWallet = KeyWallet.create();

        LOG.infoEntering("deploy", "SCORE1");
        StepCounterScore score1 = StepCounterScore.mustDeploy(txHandler, ownerWallet);
        LOG.infoExiting("deployed:" + score1);
        LOG.infoEntering("deploy", "SCORE2");
        StepCounterScore score2 = StepCounterScore.mustDeploy(txHandler, ownerWallet);
        LOG.infoExiting("deployed:" + score2);

        TransactionResult txr;
        BigInteger v1, v2, v, v1new, v2new;

        LOG.infoEntering("call", score1 + ".getStep()");
        v1 = score1.getStep(ownerWallet.getAddress());
        LOG.infoExiting(v1.toString());
        LOG.infoEntering("call", score2 + ".getStep()");
        v2 = score2.getStep(ownerWallet.getAddress());
        LOG.infoExiting(v2.toString());

        v = v1.add(BigInteger.ONE);

        LOG.infoEntering("call", score2 + ".setStepOf(" + score1 + "," + v + ")");
        txr = score2.setStepOf(ownerWallet, score1.getAddress(), v);
        assertSuccess(txr);
        LOG.infoExiting();

        v1 = score1.getStep(ownerWallet.getAddress());
        assertEquals(v, v1);

        LOG.infoEntering("call", score2 + ".setStepOf(" + score1 + "," + v + ")");
        txr = score2.setStepOf(ownerWallet, score1.getAddress(), v);
        assertFailure(txr);
        LOG.infoExiting();

        LOG.infoEntering("call", score1 + ".getStep()");
        v1 = score1.getStep(ownerWallet.getAddress());
        LOG.infoExiting(v1.toString());
        LOG.infoEntering("call", score2 + ".getStep()");
        v2 = score2.getStep(ownerWallet.getAddress());
        LOG.infoExiting(v2.toString());

        v = v.add(BigInteger.ONE);

        LOG.infoEntering("call", score1 + ".trySetStepWith(" + score2 + "," + v + ")");
        txr = score1.trySetStepWith(ownerWallet, score2.getAddress(), v);
        assertSuccess(txr);
        LOG.infoExiting();

        LOG.infoEntering("call", score2 + ".getStep()");
        v2new = score2.getStep(ownerWallet.getAddress());
        assertEquals(v2, v2new);
        LOG.infoExiting(v2new.toString());

        LOG.infoEntering("call", score1 + ".getStep()");
        v1new = score1.getStep(ownerWallet.getAddress());
        assertEquals(v, v1new);
        LOG.infoExiting(v1new.toString());
    }
}
