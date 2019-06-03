/*
 * Copyright 2018 ICON Foundation
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
 *
 */

package foundation.icon.icx.transport.jsonrpc;

import com.fasterxml.jackson.core.JsonGenerator;
import com.fasterxml.jackson.databind.JsonSerializer;
import com.fasterxml.jackson.databind.SerializerProvider;

import java.io.IOException;

/**
 * Serializers for jsonrpc value
 */
public class RpcItemSerializer extends JsonSerializer<RpcItem> {

    @Override
    public void serialize(
            RpcItem item, JsonGenerator gen, SerializerProvider serializers)
            throws IOException {
        serialize(item, gen);
    }

    private void serialize(RpcItem item, JsonGenerator gen)
            throws IOException {

        if (item instanceof RpcObject) {
            RpcObject object = item.asObject();
            gen.writeStartObject();
            for (String key : object.keySet()) {
                RpcItem value = object.getItem(key);
                if (value != null) {
                    gen.writeFieldName(key);
                    serialize(value, gen);
                }
            }
            gen.writeEndObject();
        } else if (item instanceof RpcArray) {
            RpcArray array = item.asArray();
            gen.writeStartArray();
            for (RpcItem childItem : array) {
                serialize(childItem, gen);
            }
            gen.writeEndArray();
        } else {
            gen.writeString(item.asString());
        }
    }

}
