/*
 * Copyright 2022 ICON Foundation
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

import foundation.icon.ee.test.SimpleTest;
import org.junit.jupiter.api.Test;
import score.Context;
import score.annotation.External;

public class CodecTest3 extends SimpleTest {
    public static class Score {
        @External
        public void readLess() {
            var w = Context.newByteArrayObjectWriter("RLPn");
            w.beginList(2);
            w.writeListOf(1, "str");
            w.writeListOf(1, "str");
            w.end();

            var r = Context.newByteArrayObjectReader("RLPn", w.toByteArray());
            r.beginList();
            r.beginList();
            var i = r.readInt();
            Context.require(i==1);
            r.end();

            r.beginList();
            var i2 = r.readInt();
            Context.require(i2==1);
            r.end();
            r.end();
        }

        @External
        public void readIfHasNext() {
            var w = Context.newByteArrayObjectWriter("RLPn");
            w.beginList(2);
            w.writeListOf(1, "str");
            w.writeListOf(1, "str");
            w.end();

            var r = Context.newByteArrayObjectReader("RLPn", w.toByteArray());
            r.beginList();
            r.beginList();
            var i = r.readInt();
            Context.require(i == 1);
            var s = r.readString();
            Context.require(s.equals("str"));
            if (r.hasNext()) {
                r.readString();
            }
            r.end();

            r.beginList();
            var i2 = r.readInt();
            Context.require(i2 == 1);
            var s2 = r.readString();
            Context.require(s2.equals("str"));
            r.end();
            r.end();
        }
    }

    @Test
    void readLess() {
        var c = sm.mustDeploy(Score.class);
        c.invoke("readLess");
    }

    @Test
    void readIfHasNext() {
        var c = sm.mustDeploy(Score.class);
        c.invoke("readIfHasNext");
    }
}
