/*
 * Copyright 2023 ICON Foundation
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

package foundation.icon.ee;

import foundation.icon.ee.test.NoDebugTest;
import foundation.icon.ee.types.Status;
import org.junit.jupiter.api.Assertions;
import org.junit.jupiter.api.Test;
import score.annotation.External;

public class EnumTest extends NoDebugTest {
    public static class EnumScore {
        public enum Type {
            BTP(0),
            BMC(10),
            BMV(25),
            BSH(40);

            private final int offset;

            Type(int offset) {
                this.offset = offset;
            }

            static Type valueOf(int code) throws IllegalArgumentException {
                if (code < 0) {
                    throw new IllegalArgumentException();
                }
                Type prev = null;
                for (Type v : values()) {
                    if (prev != null && code < v.offset) {
                        return prev;
                    }
                    prev = v;
                }
                return prev;
            }
        }

        @External(readonly=true)
        public String getTypeName(int code) {
            Type e = Type.valueOf(code);
            if (e.compareTo(Type.BMC) < 0) {
                return Type.BTP.toString();
            }
            return e.toString();
        }

        @External(readonly=true)
        public String[] sort(int[] items) {
            Type[] types = new Type[items.length];
            for (int i = 0; i < items.length; i++) {
                types[i] = Type.valueOf(items[i]);
            }
            sort(types);

            String[] ret = new String[types.length];
            for (int i = 0; i < ret.length; i++) {
                ret[i] = types[i].name();
            }
            return ret;
        }

        static <T extends Comparable<T>> void sort(T[] a) {
            int len = a.length;
            for (int i = 0; i < len; i++) {
                T v = a[i];
                for (int j = i+1; j < len; j++) {
                    if (v.compareTo(a[j]) > 0) {
                        T t = v;
                        v = a[j];
                        a[j] = t;
                    }
                }
                a[i] = v;
            }
        }
    }

    @Test
    void enumCompareTo() {
        var c = sm.mustDeploy(EnumScore.class);
        Assertions.assertEquals("BTP", c.query("getTypeName", 0).getRet());
        Assertions.assertEquals("BTP", c.query("getTypeName", 9).getRet());
        Assertions.assertEquals("BMC", c.query("getTypeName", 10).getRet());

        var res = c.query("sort", (Object) new Integer[]{40, 25, 10, 0});
        Assertions.assertEquals(Status.Success, res.getStatus());
        var expected = new String[]{"BTP", "BMC", "BMV", "BSH"};
        Assertions.assertArrayEquals(expected, (Object[]) res.getRet());
    }
}
