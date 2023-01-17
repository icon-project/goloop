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

package foundation.icon.ee;

import foundation.icon.ee.test.ContractAddress;
import foundation.icon.ee.test.Matcher;
import foundation.icon.ee.test.NoDebugTest;
import org.junit.jupiter.api.Test;
import score.Context;
import score.annotation.External;

import java.util.Map;

public class ThrowableMessageTest extends NoDebugTest {
    public static class MyException extends RuntimeException {
        MyException(String msg) {
            super(msg);
        }
    }

    public static class Score {
        private final static String myMessage = "my message";
        private Throwable fromSystem;
        private Throwable fromUser;

        private void throwSystemException() throws ArrayIndexOutOfBoundsException {
        }

        @External
        public void setFromSystem() {
            try {
                int[] arr = new int[0];
                var a = arr[10];
            } catch(Throwable t) {
                fromSystem = t;
                t.printStackTrace();
                Context.require(t.getMessage() == null);
                Context.require(t.toString().equals(
                        ArrayIndexOutOfBoundsException.class.getName()
                ));
            }
        }

        @External
        public void setFromUser() {
            try {
                throw new ArrayIndexOutOfBoundsException(myMessage);
            } catch (Throwable t) {
                fromUser = t;
                t.printStackTrace();
                Context.require(t.getMessage().equals(myMessage));
                Context.require(t.toString().equals(
                        ArrayIndexOutOfBoundsException.class.getName()
                                + ": " + myMessage
                ));
            }
        }

        @External
        public void checkFromSystem() {
            var t = fromSystem;
            t.printStackTrace();
            Context.require(t.getMessage() == null);
            Context.require(t.toString().equals(
                    ArrayIndexOutOfBoundsException.class.getName()
            ));
        }

        @External
        public void checkFromUser() {
            var t = fromUser;
            t.printStackTrace();
            Context.require(t.getMessage().equals(myMessage));
            Context.require(t.toString().equals(
                    ArrayIndexOutOfBoundsException.class.getName()
                            + ": " + myMessage
            ));
        }
    }

    void invokeAndMatch(ContractAddress c, String method, Matcher matcher) {
        sm.setLogger(matcher);
        c.invoke(method);
        matcher.assertOK();
    }

    @Test
    public void test() {
        createAndAcceptNewJAVAEE();
        var score = sm.mustDeploy(new Class[]{Score.class, MyException.class});

        invokeAndMatch(score, "setFromSystem", new Matcher(Map.of(
                "s.java.lang.ArrayIndexOutOfBoundsException: : Index 10 out of bounds for length 0", true,
                "at u.foundation.icon.ee.ThrowableMessageTest$Score.avm_setFromSystem", true
        )));

        // to run without object cache
        sm.setIndexer(addr -> 1);
        invokeAndMatch(score, "checkFromSystem", new Matcher(Map.of(
                "s.java.lang.ArrayIndexOutOfBoundsException: : Index 10 out of bounds for length 0", false,
                "at u.foundation.icon.ee.ThrowableMessageTest$Score.avm_setFromSystem", false,
                "s.java.lang.ArrayIndexOutOfBoundsException: : ", true
        )));

        sm.setIndexer(addr -> 0);
        invokeAndMatch(score, "setFromUser", new Matcher(Map.of(
                "s.java.lang.ArrayIndexOutOfBoundsException: my message: ", true,
                "at u.foundation.icon.ee.ThrowableMessageTest$Score.avm_setFromUser", true
        )));

        sm.setIndexer(addr -> 1);
        invokeAndMatch(score, "checkFromUser", new Matcher(Map.of(
                "s.java.lang.ArrayIndexOutOfBoundsException: my message: ", true,
                "at u.foundation.icon.ee.ThrowableMessageTest$Score.avm_setFromUser", false
        )));
    }
}
