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

import foundation.icon.icx.data.BlockNotification;
import foundation.icon.icx.transport.jsonrpc.RpcArray;
import foundation.icon.icx.transport.jsonrpc.RpcObject;
import foundation.icon.icx.transport.jsonrpc.RpcValue;

import java.math.BigInteger;

public class BlockMonitorSpec extends MonitorSpec<BlockNotification> {
    private final BigInteger height;
    private final EventMonitorSpec.EventFilter[] eventFilters;

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
            RpcArray.Builder arrBuilder = new RpcArray.Builder();
            for (EventMonitorSpec.EventFilter ef : this.eventFilters) {
                RpcObject.Builder efBuilder = new RpcObject.Builder();
                ef.apply(efBuilder);
                arrBuilder.add(efBuilder.build());
            }
            builder.put("eventFilters", arrBuilder.build());
        }
        return builder.build();
    }

    @Override
    public Class<BlockNotification> getNotificationClass() {
        return BlockNotification.class;
    }
}
