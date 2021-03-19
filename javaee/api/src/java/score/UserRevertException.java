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

package score;

/**
 *  Signals failure of an {@link score.annotation.External} method. If an
 *  external method is called by an external transaction or an internal
 *  transaction and the method throws an exception of this class or subclass of
 *  this class during the execution, the transaction fails and the code returned
 *  by {@link #getCode} is used as user failure code of the transaction. If the
 *  code is out of range, the code is clamped.
 *
 *  <p>User may extend this class and override {@link #getCode} method.
 */
public class UserRevertException extends RuntimeException {
    public UserRevertException() {
    }

    public UserRevertException(String message) {
        super(message);
    }

    public UserRevertException(String message, Throwable cause) {
        super(message, cause);
    }

    public UserRevertException(Throwable cause) {
        super(cause);
    }

    /**
     * Returns error code. Subclass may override this method to change failure
     * code. Default implementation returns {@code 0}.
     * @return error code.
     */
    public int getCode() {
        return 0;
    }
}
