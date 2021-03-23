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
import foundation.icon.ee.test.TransactionException;
import foundation.icon.ee.types.Status;
import org.junit.jupiter.api.Assertions;
import org.junit.jupiter.api.Test;
import score.ByteArrayObjectWriter;
import score.Context;
import score.ObjectReader;
import score.RevertedException;
import score.annotation.External;

public class SimpleCodecTest extends SimpleTest {
    public static class Score {
        private ObjectReader r;
        private ByteArrayObjectWriter w;

        @External
        public void memberReaderBeforeCall() {
            r = Context.newByteArrayObjectReader("RLPn", new byte[0]);
            Context.call(Context.getAddress(), "dummyMethod");
            // shall not reach here
            Context.revert();
        }

        @External
        public void memberWriterBeforeCall() {
            w = Context.newByteArrayObjectWriter("RLPn");
            Context.call(Context.getAddress(), "dummyMethod");
            // shall not reach here
            Context.revert();
        }

        @External
        public void memberReaderBeforeReturn() {
            r = Context.newByteArrayObjectReader("RLPn", new byte[0]);
        }

        @External
        public void memberWriterBeforeReturn() {
            w = Context.newByteArrayObjectWriter("RLPn");
        }

        @External
        public void localReaderBeforeCall() {
            var r = Context.newByteArrayObjectReader("RLPn", new byte[0]);
            Context.call(Context.getAddress(), "dummyMethod");
        }

        @External
        public void localWriterBeforeCall() {
            var w = Context.newByteArrayObjectWriter("RLPn");
            Context.call(Context.getAddress(), "dummyMethod");
        }

        @External
        public void localReaderBeforeReturn() {
            var r = Context.newByteArrayObjectReader("RLPn", new byte[0]);
        }

        @External
        public void localWriterBeforeReturn() {
            var w = Context.newByteArrayObjectWriter("RLPn");
        }

        @External
        public void dummyMethod() {
        }
    }

    @Test
    void beforeCallAndReturn() {
        var s = sm.mustDeploy(Score.class);
        var e = Assertions.assertThrows(TransactionException.class,
                () -> s.invoke("memberReaderBeforeCall"));
        Assertions.assertEquals(Status.IllegalObjectGraph, e.getResult().getStatus());
        e = Assertions.assertThrows(TransactionException.class,
                () -> s.invoke("memberWriterBeforeCall"));
        Assertions.assertEquals(Status.IllegalObjectGraph, e.getResult().getStatus());
        e = Assertions.assertThrows(TransactionException.class,
                () -> s.invoke("memberReaderBeforeReturn"));
        Assertions.assertEquals(Status.IllegalObjectGraph, e.getResult().getStatus());
        e = Assertions.assertThrows(TransactionException.class,
                () -> s.invoke("memberWriterBeforeReturn"));
        Assertions.assertEquals(Status.IllegalObjectGraph, e.getResult().getStatus());
        s.invoke("localReaderBeforeCall");
        s.invoke("localReaderBeforeCall");
        s.invoke("localWriterBeforeReturn");
        s.invoke("localWriterBeforeReturn");
    }

    public static class ReaderInInit {
        private ObjectReader r = Context.newByteArrayObjectReader("RLPn", new byte[0]);

        @External
        public void dummyMethod() {
        }
    }

    @Test
    void readerInInit() {
        Assertions.assertThrows(TransactionException.class,
                () -> sm.mustDeploy(ReaderInInit.class));
    }

    public static class Deployer {
        @External
        public boolean deploy(byte[] jar) {
            try {
                Context.deploy(jar);
            } catch (RevertedException e) {
                return false;
            }
            return true;
        }
    }

    @Test
    void readInInitInternalTX() {
        var s = sm.mustDeploy(Deployer.class);
        var jar = makeRelJar(ReaderInInit.class);
        var res = s.invoke("deploy", jar);
        Assertions.assertEquals(false, res.getRet());
    }

    public static class ContextLevelCodecUser {
        @External
        public void f() {
            var enc = Context.newByteArrayObjectWriter("RLPn");
            enc.write(10);
            enc.write("a string");
            var dec = Context.newByteArrayObjectReader("RLPn", enc.toByteArray());
            Context.require(dec.readInt() == 10);
            Context.require( dec.readString().equals("a string"));
        }
    }

    @Test
    public void contextLevelCodec() {
        var score = sm.mustDeploy(ContextLevelCodecUser.class);
        score.invoke("f");
    }
}
