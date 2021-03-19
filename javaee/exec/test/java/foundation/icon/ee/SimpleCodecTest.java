/*
 * Copyright 2021 ICON Foundation
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
import score.ByteArrayObjectWriter;
import score.Context;
import score.ObjectReader;
import score.annotation.External;

public class SimpleCodecTest extends SimpleTest {
    public static class RWHolder {
        private ObjectReader r;
        private ByteArrayObjectWriter w;

        @External
        public void setupRW(byte[] bytes) {
            r = Context.newByteArrayObjectReader("RLPn", bytes);
            w = Context.newByteArrayObjectWriter("RLPn");
            w.write(bytes);
        }

        @External
        public void useReader() {
            try {
                r.hasNext();
                Context.revert();
            } catch (IllegalStateException e) {
                // expected
            }
        }

        @External
        public void useWriter() {
            try {
                w.write(0);
                Context.revert();
            } catch (IllegalStateException e) {
                // expected
            }
        }
    }

    @Test
    public void expectIllegalStateExceptionForInvalidatedReaderWriter() {
        var score = sm.mustDeploy(RWHolder.class);
        var by = new byte[1000];
        score.invoke("setupRW", by);
        score.invoke("useReader");
        score.invoke("useWriter");
    }

    @Test
    public void deserializeReaderWriter() {
        var score = sm.mustDeploy(RWHolder.class);
        var by = new byte[1000];
        score.invoke("setupRW", by);
        // to run without cache
        createAndAcceptNewJAVAEE();
        sm.setIndexer((addr) -> 1);
        score.invoke("useReader");
        score.invoke("useWriter");
    }
}
