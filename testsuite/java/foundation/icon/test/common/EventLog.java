package foundation.icon.test.common;

import foundation.icon.icx.data.TransactionResult;
import foundation.icon.icx.transport.jsonrpc.RpcItem;
import foundation.icon.icx.transport.jsonrpc.RpcValue;

import java.util.ArrayList;
import java.util.Collection;
import java.util.Iterator;
import java.util.List;

public class EventLog {
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
        var items = new ArrayList<RpcItem>();
        items.addAll(log.getIndexed());
        var data = log.getData();
        if (data != null) {
            items.addAll(data);
        }

        for (int idx = 0 ; idx<params.length ; idx++) {
            if (params[idx] == null) continue;
            var item = items.get(idx);
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
}
