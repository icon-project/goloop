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
import score.Context;
import score.UserRevertException;
import score.annotation.External;

public class RevertTest extends SimpleTest {
    private void assertDeployResultStatus(int code, Class<?> cls) {
        var e = Assertions.assertThrows(
                TransactionException.class,
                ()-> sm.mustDeploy(
                        new Class<?>[] {cls, MyRevertException.class}
                )
        );
        Assertions.assertEquals(code, e.getResult().getStatus());
    }

    private void assertDeployAndInvokeResultStatus(int code, Class<?> cls) {
        var e = Assertions.assertThrows(
                TransactionException.class,
                ()-> {
                    var c = sm.mustDeploy(
                            new Class<?>[] {cls, MyRevertException.class}
                    );
                    c.invoke("f");
                }
        );
        Assertions.assertEquals(code, e.getResult().getStatus());
    }

    public static class MyRevertException extends UserRevertException {
        private final int code;

        public MyRevertException() {
            this(0);
        }

        public MyRevertException(int code) {
            this.code = code;
        }

        @Override
        public int getCode() {
            return code;
        }
    }

    public static class RevertInClinit {
        static {
            Context.revert();
        }

        @External
        public void f() {
        }
    }

    @Test
    void revertInClinit() {
        assertDeployResultStatus(Status.UserReversionStart, RevertInClinit.class);
    }

    public static class UserRevertExceptionInClinit {
        static {
            if (true) {
                throw new MyRevertException();
            }
        }

        @External
        public void f() {
        }
    }

    @Test
    void userRevertExceptionInClinit() {
        assertDeployResultStatus(Status.UserReversionStart, UserRevertExceptionInClinit.class);
    }

    public static class RevertInInit {
        public RevertInInit() {
            Context.revert();
        }

        @External
        public void f() {
        }
    }

    @Test
    void revertInInit() {
        assertDeployResultStatus(Status.UserReversionStart, RevertInInit.class);
    }

    public static class UserRevertExceptionInInit {
        public UserRevertExceptionInInit() {
            throw new MyRevertException();
        }

        @External
        public void f() {
        }
    }

    @Test
    void userRevertExceptionInInit() {
        assertDeployResultStatus(Status.UserReversionStart, UserRevertExceptionInInit.class);
    }

    public static class RevertInF {
        @External
        public void f() {
            Context.revert();
        }
    }

    @Test
    void revertInF() {
        assertDeployAndInvokeResultStatus(Status.UserReversionStart, RevertInF.class);
    }

    public static class UserRevertExceptionInF {
        @External
        public void f() {
            throw new MyRevertException();
        }
    }

    @Test
    void userRevertExceptionInF() {
        assertDeployAndInvokeResultStatus(Status.UserReversionStart, UserRevertExceptionInF.class);
    }

    public static class ValidCodeRevert {
        @External
        public void f() {
            Context.revert(10);
        }
    }

    @Test
    void validCodeRevert() {
        assertDeployAndInvokeResultStatus(Status.UserReversionStart + 10, ValidCodeRevert.class);
    }

    public static class ValidCodeUserRevertException {
        @External
        public void f() {
            throw new MyRevertException(10);
        }
    }

    @Test
    void validCodeUserRevertException() {
        assertDeployAndInvokeResultStatus(Status.UserReversionStart + 10, ValidCodeUserRevertException.class);
    }

    public static class TooLowCodeRevert {
        @External
        public void f() {
            Context.revert(-1);
        }
    }

    @Test
    void tooLowCodeRevert() {
        assertDeployAndInvokeResultStatus(Status.UserReversionStart, TooLowCodeRevert.class);
    }

    public static class TooHighCodeRevert {
        @External
        public void f() {
            Context.revert(10000);
        }
    }

    @Test
    void tooHighCodeRevert() {
        assertDeployAndInvokeResultStatus(Status.UserReversionEnd - 1, TooHighCodeRevert.class);
    }

    public static class TooLowCodeUserRevertException {
        @External
        public void f() {
            throw new MyRevertException(-1);
        }
    }

    @Test
    void tooLowCodeUserRevertException() {
        assertDeployAndInvokeResultStatus(Status.UserReversionStart, TooLowCodeUserRevertException.class);
    }

    public static class TooHighCodeUserRevertException {
        @External
        public void f() {
            throw new MyRevertException(10000);
        }
    }

    @Test
    void tooHighCodeUserRevertException() {
        assertDeployAndInvokeResultStatus(Status.UserReversionEnd-1, TooHighCodeUserRevertException.class);
    }
}
