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

import com.fasterxml.jackson.annotation.JsonInclude;
import com.fasterxml.jackson.core.JsonProcessingException;
import com.fasterxml.jackson.databind.ObjectMapper;
import com.fasterxml.jackson.databind.module.SimpleModule;
import foundation.icon.icx.Provider;
import foundation.icon.icx.Request;
import foundation.icon.icx.transport.jsonrpc.RpcConverter;
import foundation.icon.icx.transport.jsonrpc.RpcError;
import foundation.icon.icx.transport.jsonrpc.RpcItem;
import foundation.icon.icx.transport.jsonrpc.RpcItemDeserializer;
import foundation.icon.icx.transport.jsonrpc.RpcItemSerializer;
import foundation.icon.icx.transport.monitor.Monitor;
import foundation.icon.icx.transport.monitor.MonitorSpec;
import okhttp3.MediaType;
import okhttp3.OkHttpClient;
import okhttp3.RequestBody;
import okhttp3.Response;
import okhttp3.WebSocket;
import okhttp3.WebSocketListener;
import okio.BufferedSink;

import java.io.IOException;
import java.net.URI;
import java.net.URISyntaxException;

/**
 * The {@code HttpProvider} class transports JSON-RPC payloads through HTTP.
 */
public class HttpProvider implements Provider {

    private final OkHttpClient httpClient;
    private final String url;

    /**
     * Initializes a new {@code HttpProvider} with the custom http client object and the given endpoint url.
     *
     * @param httpClient a custom http client to send HTTP requests and read their responses
     * @param url an endpoint url, ex) {@code http://localhost:9000/api/v3}
     */
    public HttpProvider(OkHttpClient httpClient, String url) {
        this.httpClient = httpClient;
        try {
            new URI(url);
            this.url = url;
        } catch (URISyntaxException e) {
            throw new IllegalArgumentException(e.getMessage());
        }
    }

    /**
     * Initializes a new {@code HttpProvider} with the given endpoint url.
     * This will use a default http client object for the operation.
     *
     * @param url an endpoint url, ex) {@code http://localhost:9000/api/v3}
     */
    public HttpProvider(String url) {
        this(new OkHttpClient.Builder().build(), url);
    }

    /**
     * @see Provider#request(foundation.icon.icx.transport.jsonrpc.Request, RpcConverter)
     */
    @Override
    public <T> Request<T> request(final foundation.icon.icx.transport.jsonrpc.Request request, RpcConverter<T> converter) {

        // Makes the request body
        RequestBody body = new RequestBody() {
            @Override
            public MediaType contentType() {
                return MediaType.parse("application/json");
            }

            @Override
            public void writeTo(BufferedSink sink) throws IOException {
                ObjectMapper mapper = new ObjectMapper();
                mapper.setSerializationInclusion(JsonInclude.Include.NON_NULL);
                SimpleModule module = new SimpleModule();
                module.addSerializer(RpcItem.class, new RpcItemSerializer());
                mapper.registerModule(module);
                mapper.writeValue(sink.outputStream(), request);
            }
        };

        okhttp3.Request httpRequest = new okhttp3.Request.Builder()
                .url(url)
                .post(body)
                .build();

        return new HttpCall<>(httpClient.newCall(httpRequest), converter);
    }

    private enum WsState {
        WS_INIT,
        WS_REQUEST,
        WS_CONNECT,
        WS_START,
        WS_STOP
    }

    private class HttpMonitor<T> implements Monitor<T> {
        Monitor.Listener<T> listener;
        MonitorSpec spec;
        WsState state = WsState.WS_INIT;
        okhttp3.WebSocket ws;
        final Object condVar = new Object();
        RpcConverter<T> rpcConverter;
        ObjectMapper mapper;

        HttpMonitor(MonitorSpec spec, RpcConverter<T> converter) {
            this.spec = spec;
            this.rpcConverter = converter;

            mapper = new ObjectMapper();
            SimpleModule module = new SimpleModule();
            module.addDeserializer(RpcItem.class, new RpcItemDeserializer());
            mapper.registerModule(module);
        }

        private class WebSocketListenerImpl extends WebSocketListener {
            private final String request;
            WebSocketListenerImpl(String request) {
                this.request = request;
            }

            @Override
            public void onOpen(okhttp3.WebSocket webSocket, Response response) {
                super.onOpen(webSocket, response);
                synchronized (condVar) {
                    state = WsState.WS_CONNECT;
                }
                webSocket.send(request);
            }

            @Override
            public void onMessage(okhttp3.WebSocket webSocket, String message) {
                super.onMessage(webSocket, message);
                synchronized (condVar) {
                    switch(state) {
                        case WS_CONNECT:
                            try {
                                RpcError error = mapper.readValue(message, RpcError.class);
                                if (error.getCode() == 0) {
                                    state = WsState.WS_START;
                                    listener.onStart();
                                } else {
                                    listener.onError(error.getCode());
                                }
                            }
                            catch (IOException ex) {
                                listener.onError(100);
                            }
                            condVar.notify();
                            break;
                        case WS_START:
                            try {
                                RpcItem rpcItem = mapper.readValue(message, RpcItem.class);
                                T obj = rpcConverter.convertTo(rpcItem.asObject());
                                listener.onEvent(obj);
                            }
                            catch (IOException ex) {
                                listener.onError(100);
                            }
                            break;
                        default:
                            break;
                    }
                }
            }

            @Override
            public void onFailure(WebSocket webSocket, Throwable t, Response response) {
                listener.onError(0);
            }

            @Override
            public void onClosed(okhttp3.WebSocket webSocket, int code, String reason) {
                listener.onClose();
            }
        }

        private okhttp3.WebSocket newWebSocket(String request) {
            okhttp3.Request httpRequest = new okhttp3.Request.Builder()
                    .url(url + "/" + spec.getPath())
                    .build();
            return httpClient.newWebSocket(httpRequest, new WebSocketListenerImpl(request));
        }

        @Override
        public boolean start(Listener<T> listener) {
            synchronized (condVar) {
                switch(state) {
                    case WS_INIT:
                    case WS_STOP:
                        state = WsState.WS_REQUEST;
                        break;
                    default:
                        throw new IllegalStateException();
                }
            }
            this.listener = listener;
            ObjectMapper mapper = new ObjectMapper();
            mapper.setSerializationInclusion(JsonInclude.Include.NON_EMPTY);
            SimpleModule module = new SimpleModule();
            module.addSerializer(RpcItem.class, new RpcItemSerializer());
            mapper.registerModule(module);

            String request;
            try {
                request = mapper.writeValueAsString(spec.getParams());
            }
            catch (JsonProcessingException ex) {
                throw new IllegalArgumentException();
            }

            ws = newWebSocket(request);
            try {
                synchronized (condVar) {
                    condVar.wait(3000);
                }
            } catch (InterruptedException ex) {
                throw new IllegalStateException();
            }
            return state == WsState.WS_START;
        }

        @Override
        public void stop() {
            synchronized (condVar) {
                switch(state) {
                    case WS_INIT:
                    case WS_STOP:
                        throw new IllegalStateException(state.toString());
                    default:
                        ws.close(1000, null);
                        ws = null;
                        state = WsState.WS_STOP;
                        break;
                }
            }
        }
    }

    @Override
    public <T> Monitor<T> monitor(MonitorSpec spec, RpcConverter<T> converter) {
        return new HttpMonitor<>(spec, converter);
    }
}
