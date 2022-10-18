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

package foundation.icon.ee;

import foundation.icon.ee.ipc.EEProxy;
import foundation.icon.ee.test.SimpleTest;
import foundation.icon.ee.types.Status;
import org.aion.avm.core.IExternalState;
import org.junit.jupiter.api.Assertions;
import org.junit.jupiter.api.Test;
import score.Context;
import score.annotation.External;

import java.util.Collection;
import java.util.List;
import java.util.Map;

public class MapTest extends SimpleTest {
    public static class Score {
        private final Map<String, String> map = Map.of("k1", "v1", "k2", "v2", "k3", "v3");

        private boolean isEquivalent(Collection<String> l, Collection<String> r) {
            if (l==null && r==null) {
                return true;
            }
            if (l==null || r==null) {
                return false;
            }
            if (l.size() != r.size()) {
                return false;
            }
            return (l.containsAll(r) && r.containsAll(l));
        }

        @External
        public void valuesInNotFixMapValues() {
            Context.require(isEquivalent(map.values(), List.of("k1", "k2", "k3")));
        }

        @External
        public void valuesInFixMapValues() {
            Context.require(isEquivalent(map.values(), List.of("v1", "v2", "v3")));
        }

        @External
        public void keySet() {
            Context.require(isEquivalent(map.keySet(), List.of("k1", "k2", "k3")));
        }
    }

    @Test
    void valuesInNotFixMapValues() {
        sm.unsetRevisionFlag(IExternalState.REVISION_FIX_MAP_VALUES);
        var c = sm.mustDeploy(Score.class);
        Assertions.assertEquals(Status.Success, c.invoke("valuesInNotFixMapValues").getStatus());
    }

    @Test
    void valuesInFixMapValues() {
        sm.setRevisionFlag(IExternalState.REVISION_FIX_MAP_VALUES);
        var c = sm.mustDeploy(Score.class);
        Assertions.assertEquals(Status.Success, c.invoke("valuesInFixMapValues").getStatus());
    }

    @Test
    void keySetInNotFixMapValues() {
        sm.unsetRevisionFlag(IExternalState.REVISION_FIX_MAP_VALUES);
        var c = sm.mustDeploy(Score.class);
        Assertions.assertEquals(Status.Success, c.invoke("keySet").getStatus());
    }

    @Test
    void keySetInFixMapValues() {
        sm.setRevisionFlag(IExternalState.REVISION_FIX_MAP_VALUES);
        var c = sm.mustDeploy(Score.class);
        Assertions.assertEquals(Status.Success, c.invoke("keySet").getStatus());
    }
}
