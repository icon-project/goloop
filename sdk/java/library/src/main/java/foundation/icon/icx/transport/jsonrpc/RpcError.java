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

import java.io.IOException;

/**
 * RpcError defines the error that occurred during communicating through jsonrpc
 */
public class RpcError extends IOException {
    private long code;
    private String message;

    public RpcError() {
        // jackson needs a default constructor
    }

    public RpcError(long code, String message) {
        super(message);
        this.code = code;
        this.message = message;
    }

    /**
     * Returns the code of rpc error
     * @return error code
     */
    public long getCode() {
        return code;
    }

    /**
     * Returns the message of rpc error
     * @return error message
     */
    @Override
    public String getMessage() {
        return message;
    }
}
