package foundation.icon.icx.transport.monitor;

import foundation.icon.icx.data.Address;
import foundation.icon.icx.transport.jsonrpc.RpcArray;
import foundation.icon.icx.transport.jsonrpc.RpcObject;
import foundation.icon.icx.transport.jsonrpc.RpcValue;
import foundation.icon.icx.transport.monitor.MonitorSpec;

import java.math.BigInteger;

public class EventMonitorSpec extends MonitorSpec {
    private BigInteger height;
    private EventFilter filter;

    public static class EventFilter {
        private String event;
        private Address addr;
        private String[] indexed;
        private String[] data;
        public EventFilter(String event, Address addr, String[] indexed, String[] data) {
            this.event = event;
            this.addr = addr;
            if(indexed != null && indexed.length > 0) {
                this.indexed = new String[indexed.length];
                System.arraycopy(indexed, 0, this.indexed, 0, indexed.length);
            }
            if(data != null && data.length > 0) {
                this.data = new String[data.length];
                System.arraycopy(data, 0, this.data, 0, data.length);
            }
        }
        public void apply(RpcObject.Builder builder) {
            builder.put("event", new RpcValue(event));
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
            if (this.indexed != null) {
                RpcArray.Builder arrayBuilder = new RpcArray.Builder();
                for(String d : this.indexed) {
                    arrayBuilder.add(new RpcValue(d));
                }
                builder.put("indexed", arrayBuilder.build());
            }
        }
    }


    /**
     *
     * @param height
     * @param event
     * @param addr
     * @param data
     */
    public EventMonitorSpec(BigInteger height, String event, Address addr, String[] indexed, String[] data) {
        this.path = "event";

        this.height = height;
        this.filter = new EventFilter(event, addr, indexed, data);
    }

    @Override
    public RpcObject getParams() {
        RpcObject.Builder builder = new RpcObject.Builder()
                .put("height", new RpcValue(height));
                this.filter.apply(builder);
        return builder.build();
    }
}
