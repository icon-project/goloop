/*
 * Copyright 2019 ICON Foundation
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

package foundation.icon.icx.transport.monitor;

import foundation.icon.icx.data.Address;
import foundation.icon.icx.data.EventNotification;
import foundation.icon.icx.transport.jsonrpc.RpcArray;
import foundation.icon.icx.transport.jsonrpc.RpcObject;
import foundation.icon.icx.transport.jsonrpc.RpcValue;

import java.math.BigInteger;

public class EventMonitorSpec extends MonitorSpec<EventNotification> {
    private final BigInteger height;
    private final EventFilter[] filters;
    private final boolean logs;
    private final long progressInterval;

    public static class EventFilter {
        private final String event;
        private final Address addr;
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

    public EventMonitorSpec(BigInteger height, String event, Address addr, String[] indexed, String[] data) {
        this(height, event, addr, indexed, data, false);
    }

    public EventMonitorSpec(BigInteger height, String event, Address addr, String[] indexed, String[] data, boolean logs) {
        this(height, event, addr, indexed, data, logs, 0);
    }

    public EventMonitorSpec(BigInteger height, String event, Address addr, String[] indexed, String[] data, boolean logs, long progressInterval) {
        this(height, new EventFilter[]{new EventFilter(event, addr, indexed, data)}, logs, progressInterval);
    }

    public EventMonitorSpec(BigInteger height, EventFilter[] filters, boolean logs, long progressInterval) {
        this.path = "event";
        this.height = height;
        this.filters = filters;
        this.logs = logs;
        this.progressInterval = progressInterval;
    }

    @Override
    public RpcObject getParams() {
        RpcObject.Builder builder = new RpcObject.Builder()
                .put("height", new RpcValue(height));
        if (this.logs) {
            builder = builder.put("logs", new RpcValue(true));
        }
        if (filters.length==1) {
            this.filters[0].apply(builder);
        } else {
            RpcArray.Builder arrBuilder = new RpcArray.Builder();
            for (EventFilter ef : this.filters) {
                RpcObject.Builder efBuilder = new RpcObject.Builder();
                ef.apply(efBuilder);
                arrBuilder.add(efBuilder.build());
            }
            builder = builder.put("eventFilters", arrBuilder.build());
        }
        if (this.progressInterval>0) {
            builder = builder.put("progressInterval", new RpcValue(BigInteger.valueOf(this.progressInterval)));
        }
        return builder.build();
    }

    @Override
    public Class<EventNotification> getNotificationClass() {
        return EventNotification.class;
    }
}
