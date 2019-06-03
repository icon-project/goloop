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

package foundation.icon.icx;

import foundation.icon.icx.transport.jsonrpc.RpcConverter;

/**
 * The {@code Provider} class transports the request and receives the response.
 */
public interface Provider {

    /**
     * Prepares to execute the request
     *
     * @param request   the request to send
     * @param converter the converter for the response data
     * @param <T>       the return type
     * @return a {@code Request} object to be executed
     */
    <T> Request<T> request(foundation.icon.icx.transport.jsonrpc.Request request, RpcConverter<T> converter);
}
