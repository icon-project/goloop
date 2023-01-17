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

package foundation.icon.icx.data;

public class Base64 {
    private final String data;

    public Base64(String data) {
        this.data = data;
    }

    public byte[] decode() {
        return java.util.Base64.getDecoder().decode(data);
    }

    @Override
    public boolean equals(Object obj) {
        if (obj == this) return true;
        if (obj instanceof Base64) {
            Base64 other = (Base64) obj;
            return this.data.equals(other.data);
        }
        return false;
    }

    @Override
    public String toString() {
        return this.data;
    }
}
