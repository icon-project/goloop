package foundation.icon.icx.transport.monitor;

import foundation.icon.icx.transport.jsonrpc.RpcObject;
import foundation.icon.icx.transport.jsonrpc.RpcValue;

import java.math.BigInteger;

public class BlockMonitorSpec extends MonitorSpec {
    private BigInteger height;
    private EventMonitorSpec.EventFilter[] eventFilters;

    public BlockMonitorSpec(BigInteger height, EventMonitorSpec.EventFilter[] eventFilters) {
        this.height = height;
        this.path = "block";
        this.eventFilters = eventFilters;
    }

    @Override
    public RpcObject getParams() {
        RpcObject.Builder builder = new RpcObject.Builder()
                .put("height", new RpcValue(this.height));
        if (this.eventFilters != null) {
            for (EventMonitorSpec.EventFilter ef : this.eventFilters) {
                ef.apply(builder);
            }
        }
        return builder.build();
    }
}
