package foundation.icon.icx.data;

import foundation.icon.icx.transport.jsonrpc.RpcItem;
import foundation.icon.icx.transport.jsonrpc.RpcObject;

import java.math.BigInteger;

public class EventNotification {
    private RpcObject properties;

    EventNotification(RpcObject properties) {
        this.properties = properties;
    }

    public Bytes getHash() {
        RpcItem item = properties.getItem("hash");
        return item != null ? item.asBytes() : null;
    }

    public BigInteger getHeight() {
        RpcItem item = properties.getItem("height");
        return item != null ? item.asInteger() : null;
    }

    public BigInteger getIndex() {
        RpcItem item = properties.getItem("index");
        return item != null ? item.asInteger() : null;
    }
}
