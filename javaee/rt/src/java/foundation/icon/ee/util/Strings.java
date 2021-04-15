/*
 * Copyright 2020 ICON Foundation
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

package foundation.icon.ee.util;

public class Strings {
    public static int countPrefixRun(String str, char run) {
        int i = 0;
        for (; i<str.length(); ++i) {
            if (str.charAt(i) != run) {
                break;
            }
        }
        return i;
    }

    private static final String hexDigits = "0123456789abcdef";

    public static String hexFromBytes(byte[] ba) {
        return hexFromBytes(ba, "");
    }

    public static String hexFromBytes(byte[] ba, String sep) {
        var sb = new StringBuilder();
        for (int i=0; i<ba.length; i++) {
            if (i>0) {
                sb.append(sep);
            }
            sb.append(hexDigits.charAt((ba[i]>>4)&0xf));
            sb.append(hexDigits.charAt((ba[i])&0xf));
        }
        return sb.toString();
    }
}
