/*
 * Copyright 2023 ICON Foundation
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
 *
 */

package foundation.icon.test.cases;

import foundation.icon.icx.IconService;
import foundation.icon.icx.transport.http.HttpProvider;
import foundation.icon.test.common.Constants;
import foundation.icon.test.common.Env;
import foundation.icon.test.common.TestBase;
import org.junit.jupiter.api.Assertions;
import org.junit.jupiter.api.BeforeAll;
import org.junit.jupiter.api.Tag;
import org.junit.jupiter.api.Test;

import java.math.BigInteger;

import static foundation.icon.test.common.Env.LOG;

@Tag(Constants.TAG_JAVA_SCORE)
public class NetworkInfoTest extends TestBase {
    private static IconService iconService;
    private static Env.Channel channel;
    @BeforeAll
    static void init() {
        Env.Node node = Env.nodes[0];
        channel = node.channels[0];
        iconService = new IconService(new HttpProvider(channel.getAPIUrl(Env.testApiVer)));
    }

    @Test
    public void verifyNetworkInfo() throws Exception {
        LOG.infoEntering("verifyNetworkInfo");
        var info = iconService.getNetworkInfo().execute();
        Assertions.assertEquals(info.getChannel(), channel.name);
        Assertions.assertEquals(info.getEarliest(), BigInteger.ZERO);
        Assertions.assertEquals(info.getPlatform(), "basic");
        Assertions.assertEquals(info.getNID().intValueExact(), channel.chain.networkId);
        LOG.infoExiting();
    }
}
