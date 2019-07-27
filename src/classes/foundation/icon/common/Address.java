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

package foundation.icon.common;

import java.util.Arrays;

public class Address {
    private byte prefix;
    private byte[] body;

    public Address(byte[] input) {
        if (input.length != 21) {
            throw new IllegalArgumentException("Illegal format");
        }
        this.prefix = input[0];
        this.body = Arrays.copyOfRange(input, 1, input.length);
    }

    public String toString() {
        return ((prefix == 0) ? "hx" : "cx") + Bytes.toHexString(body);
    }

    public byte[] toByteArray() {
        byte[] ba = new byte[21];
        ba[0] = prefix;
        System.arraycopy(body, 0, ba, 1, body.length);
        return ba;
    }
}
