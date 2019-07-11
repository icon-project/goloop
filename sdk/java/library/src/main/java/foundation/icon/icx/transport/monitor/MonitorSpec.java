package foundation.icon.icx.transport.monitor;

import foundation.icon.icx.transport.jsonrpc.RpcObject;

public abstract class MonitorSpec {
    protected String path;

    public abstract RpcObject getParams();

    public String getPath() {return path;}
}
