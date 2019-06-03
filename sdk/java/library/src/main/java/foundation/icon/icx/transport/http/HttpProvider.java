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

package foundation.icon.icx.transport.http;

import com.fasterxml.jackson.annotation.JsonInclude;
import com.fasterxml.jackson.databind.ObjectMapper;
import com.fasterxml.jackson.databind.module.SimpleModule;
import foundation.icon.icx.Provider;
import foundation.icon.icx.Request;
import foundation.icon.icx.transport.jsonrpc.RpcConverter;
import foundation.icon.icx.transport.jsonrpc.RpcItem;
import foundation.icon.icx.transport.jsonrpc.RpcItemSerializer;
import okhttp3.MediaType;
import okhttp3.OkHttpClient;
import okhttp3.RequestBody;
import okio.BufferedSink;

import java.io.IOException;
import java.net.URI;
import java.net.URISyntaxException;
import java.util.HashMap;

/**
 * The {@code HttpProvider} class transports JSON-RPC payloads through HTTP.
 */
public class HttpProvider implements Provider {

    private final OkHttpClient httpClient;
    private String serverUri;
    private int version;
    private HashMap<String, String> urlMap;

    /**
     * @deprecated Do not use this constructor. Use {@link #HttpProvider(OkHttpClient, String, int)} instead.
     */
    public HttpProvider(OkHttpClient httpClient, String url) {
        this(httpClient, true, url, 3);
    }

    /**
     * Initializes a new {@code HttpProvider} with the custom http client object and the given server uri.
     *
     * @param httpClient a custom http client to send HTTP requests and read their responses
     * @param uri a server-based authority URI format, ex) {@code <scheme>://<host>[:port]}
     * @param version the version of JSON-RPC APIs
     *
     * @since 0.9.12
     */
    public HttpProvider(OkHttpClient httpClient, String uri, int version) {
        this(httpClient, false, uri, version);
    }

    /**
     * @deprecated Do not use this constructor. Use {@link #HttpProvider(String, int)} instead.
     */
    public HttpProvider(String url) {
        this(new OkHttpClient.Builder().build(), url);
    }

    /**
     * Initializes a new {@code HttpProvider} with the given server uri.
     * This will use a default http client object for the operation.
     *
     * @param uri a server-based authority URI format, ex) {@code <scheme>://<host>[:port]}
     * @param version the version of JSON-RPC APIs
     *
     * @since 0.9.12
     */
    public HttpProvider(String uri, int version) {
        this(new OkHttpClient.Builder().build(), uri, version);
    }

    private HttpProvider(OkHttpClient httpClient, boolean allowPath, String uri, int version) {
        this.httpClient = httpClient;
        if (version != 3) {
            throw new IllegalArgumentException("Unsupported version");
        }
        this.version = version;
        new Parser(uri).parse(allowPath);
        generateUrlMap();
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

        String method = request.getMethod();
        String url = urlMap.get(method.substring(0, method.indexOf("_")));

        okhttp3.Request httpRequest = new okhttp3.Request.Builder()
                .url(url)
                .post(body)
                .build();

        return new HttpCall<>(httpClient.newCall(httpRequest), converter);
    }

    private void generateUrlMap() {
        urlMap = new HashMap<>();
        urlMap.put("icx", serverUri + "/api/v" + version);
        urlMap.put("debug", serverUri + "/api/debug/v" + version);
    }

    private class Parser {

        private final String input;

        Parser(String s) {
            input = s;
        }

        void parse(boolean allowPath) {
            try {
                URI uri = new URI(input);
                if (allowPath) {
                    if (!"/api/v3".equals(uri.getPath())) {
                        throw new IllegalArgumentException("Malformed endpoint URI");
                    }
                } else {
                    if (!"".equals(uri.getPath())) {
                        throw new IllegalArgumentException("Path is not allowed");
                    }
                }
                serverUri = uri.getScheme() + "://" + uri.getAuthority();
            } catch (URISyntaxException e) {
                throw new IllegalArgumentException(e.getMessage());
            }
        }
    }
}
