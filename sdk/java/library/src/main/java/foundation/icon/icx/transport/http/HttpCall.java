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
 */

package foundation.icon.icx.transport.http;

import com.fasterxml.jackson.databind.DeserializationFeature;
import com.fasterxml.jackson.databind.ObjectMapper;
import com.fasterxml.jackson.databind.module.SimpleModule;
import foundation.icon.icx.Callback;
import foundation.icon.icx.Request;
import foundation.icon.icx.transport.jsonrpc.Response;
import foundation.icon.icx.transport.jsonrpc.RpcConverter;
import foundation.icon.icx.transport.jsonrpc.RpcError;
import foundation.icon.icx.transport.jsonrpc.RpcItem;
import foundation.icon.icx.transport.jsonrpc.RpcItemDeserializer;
import okhttp3.ResponseBody;

import java.io.IOException;

/**
 * Http call can be executed by this class
 *
 * @param <T> the data type of the response
 */
public class HttpCall<T> implements Request<T> {

    private final okhttp3.Call httpCall;
    private final RpcConverter<T> converter;

    HttpCall(okhttp3.Call httpCall, RpcConverter<T> converter) {
        this.httpCall = httpCall;
        this.converter = converter;
    }

    @Override
    public T execute() throws IOException {
        return convertResponse(httpCall.execute());
    }

    @Override
    public void execute(final Callback<T> callback) {
        httpCall.enqueue(new okhttp3.Callback() {
            @Override
            public void onFailure(okhttp3.Call call, IOException e) {
                callback.onFailure(e);
            }

            @Override
            public void onResponse(
                    okhttp3.Call call, okhttp3.Response response) {
                try {
                    T result = convertResponse(response);
                    callback.onSuccess(result);
                } catch (IOException e) {
                    callback.onFailure(e);
                }
            }
        });
    }

    // Converts the response data from the OkHttp response
    private T convertResponse(okhttp3.Response httpResponse) throws IOException {
        ResponseBody body = httpResponse.body();
        if (body != null) {
            ObjectMapper mapper = new ObjectMapper();
            mapper.configure(DeserializationFeature.FAIL_ON_UNKNOWN_PROPERTIES, false);
            mapper.registerModule(createDeserializerModule());
            String content = body.string();
            Response response = mapper.readValue(content, Response.class);
            if (converter == null) {
                throw new IllegalArgumentException("There is no converter for response:'" + content + "'");
            }
            if (response.getError() != null) {
                throw response.getError();
            }
            return converter.convertTo(response.getResult());
        } else {
            throw new RpcError(httpResponse.code(), httpResponse.message());
        }
    }

    private SimpleModule createDeserializerModule() {
        SimpleModule module = new SimpleModule();
        module.addDeserializer(RpcItem.class, new RpcItemDeserializer());
        return module;
    }
}
