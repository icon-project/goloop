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

package score;

/**
 * Signals a manual reversion from a score.
 */
public class UserRevertedException extends RevertedException {
    // NOTE: the following codes should be matched with {@code foundation.icon.ee.types.Status}
    private static final int Start = 32;
    private static final int End = 1000 - Start;

    private int statusCode;

    /**
     * Constructs a new exception
     */
    public UserRevertedException() {
        super();
    }

    /**
     * Constructs a new exception
     * @param message message
     */
    public UserRevertedException(String message) {
        super(message);
    }

    /**
     * Constructs a new exception
     * @param message message
     * @param cause cause
     */
    public UserRevertedException(String message, Throwable cause) {
        super(message, cause);
    }

    /**
     * Constructs a new exception
     * @param cause cause
     */
    public UserRevertedException(Throwable cause) {
        super(cause);
    }

    /**
     * Constructs a new exception
     * @param code reversion code defined by score
     */
    public UserRevertedException(int code) {
        super();
        statusCode = code;
    }

    /**
     * Constructs a new exception
     * @param code reversion code defined by score
     * @param message message
     */
    public UserRevertedException(int code, String message) {
        super(message);
        statusCode = code;
    }

    /**
     * Constructs a new exception
     * @param code reversion code defined by score
     * @param message message
     * @param cause cause
     */
    public UserRevertedException(int code, String message, Throwable cause) {
        super(message, cause);
        statusCode = code;
    }

    /**
     * Constructs a new exception
     * @param code reversion code defined by score
     * @param cause cause
     */
    public UserRevertedException(int code, Throwable cause) {
        super(cause);
        statusCode = code;
    }

    /**
     * Returns reversion code.
     * @return reversion code.
     */
    public int getCode() {
        return statusCode;
    }
}
