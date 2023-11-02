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

import foundation.icon.ee.test.SimpleTest;
import foundation.icon.ee.types.Status;
import org.junit.jupiter.api.Assertions;
import org.junit.jupiter.api.Test;
import score.Context;
import score.ObjectReader;
import score.ObjectWriter;
import score.annotation.External;

import java.math.BigInteger;

public class RLPTest extends SimpleTest {
    public static class Score {
        @External(readonly=true)
        public long readOverflowLenString() {
            byte[] data= new byte[] {(byte)0xbb,(byte)0xff,(byte)0xff,(byte)0xff,(byte)0xfb,0x42,0x43,0x44,0x46,0x47,0x48};
            ObjectReader r = Context.newByteArrayObjectReader("RLPn", data);
            r.skip(0x7fffffff);

            return 1;
        }

        @External
        public void skipBasic() {
            byte[] data = new byte[5];
            data[3] = 1;
            ObjectReader r = Context.newByteArrayObjectReader("RLPn", data);
            r.skip(3);
            var by = r.readByte();
            Context.require(by==1);
            r.skip(1);
        }

        @External
        public void negativeSkipCount() {
            byte[] data = new byte[] {0, 1};
            ObjectReader r = Context.newByteArrayObjectReader("RLPn", data);
            r.skip(-1);
            var by = r.read(Byte.class);
            Context.require(by==0);
        }
    }

    @Test
    void readOverflowLenString() {
        var c = sm.mustDeploy(Score.class);
        var res = c.tryInvoke("readOverflowLenString");
        Assertions.assertEquals(Status.UnknownFailure, res.getStatus());
    }

    @Test
    void skipBasic() {
        var c = sm.mustDeploy(Score.class);
        c.invoke("skipBasic");
    }

    @Test
    void negativeSkipCount() {
        var c = sm.mustDeploy(Score.class);
        c.invoke("negativeSkipCount");
    }
}
