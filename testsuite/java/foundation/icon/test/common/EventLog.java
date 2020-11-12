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

package foundation.icon.test.common;

import foundation.icon.icx.data.TransactionResult;

import java.util.ArrayList;
import java.util.List;

import static foundation.icon.test.common.Env.LOG;

public class EventLog {
    private static final boolean DEBUG = false;
    final private String score;
    final private String[] params;

    public EventLog(String score, String ...params) {
        this.score = score;
        this.params = params;
    }

    public boolean check(TransactionResult.EventLog log) {
        if (score != null && !score.equals(log.getScoreAddress())) {
            return false;
        }
        var items = new ArrayList<>(log.getIndexed());
        var data = log.getData();
        if (data != null) {
            items.addAll(data);
        }
        for (int idx = 0; idx < params.length; idx++) {
            debugInfo(String.format("params[%d] = %s", idx, params[idx]));
            if (params[idx] == null) continue;
            var item = items.get(idx);
            debugInfo(String.format("     item = %s", item != null ? item.asString() : "null"));
            if (item == null) {
                return false;
            }
            if (!params[idx].equals(item.asString())) {
                return false;
            }
        }
        return true;
    }

    public static boolean checkScenario(List<EventLog> scenario, TransactionResult result) {
        var itr = scenario.iterator();
        var seq = itr.next();
        for (var log : result.getEventLogs()) {
            if (seq.check(log)) {
                if (!itr.hasNext()) {
                    return true;
                }
                seq = itr.next();
            }
        }
        return false;
    }

    private void debugInfo(String msg) {
        if (DEBUG) {
            LOG.info(msg);
        }
    }
}
