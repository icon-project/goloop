package foundation.icon.icx.transport.monitor;

import foundation.icon.icx.data.Address;
import foundation.icon.icx.transport.jsonrpc.RpcArray;
import foundation.icon.icx.transport.jsonrpc.RpcObject;
import foundation.icon.icx.transport.jsonrpc.RpcValue;
import foundation.icon.icx.transport.monitor.MonitorSpec;

import java.math.BigInteger;

public class EventMonitorSpec extends MonitorSpec {
    private BigInteger height;
    private String event;
    private Address addr;
    private String[] data;

    /**
     *
     * @param height
     * @param event
     * @param addr
     * @param data
     */
    public EventMonitorSpec(BigInteger height, String event, Address addr, String[] data) {
        this.path = "event";

        this.height = height;
        this.event = event;
        this.addr = addr;
        if(data != null && data.length > 0) {
            this.data = new String[data.length];
            System.arraycopy(data, 0, this.data, 0, data.length);
        }
    }

    @Override
    public RpcObject getParams() {
        RpcObject.Builder builder = new RpcObject.Builder()
                .put("height", new RpcValue(height))
                .put("event", new RpcValue(event));
        if (this.addr != null) {
            builder.put("addr", new RpcValue(addr));
        }
        if (this.data != null) {
            RpcArray.Builder arrayBuilder = new RpcArray.Builder();
            for(String d : this.data) {
                arrayBuilder.add(new RpcValue(d));
            }
            builder.put("data", arrayBuilder.build());
        }
        return builder.build();
    }
}
