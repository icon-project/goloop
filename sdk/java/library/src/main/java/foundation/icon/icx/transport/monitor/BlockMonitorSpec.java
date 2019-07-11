package foundation.icon.icx.transport.monitor;

import foundation.icon.icx.transport.jsonrpc.RpcObject;
import foundation.icon.icx.transport.jsonrpc.RpcValue;

import java.math.BigInteger;

public class BlockMonitorSpec extends MonitorSpec {
    private BigInteger height;

    public BlockMonitorSpec(BigInteger height) {
        this.height = height;
        this.path = "block";
    }

    @Override
    public RpcObject getParams() {
        return new RpcObject.Builder()
                .put("height", new RpcValue(this.height))
                .build();
    }
}
