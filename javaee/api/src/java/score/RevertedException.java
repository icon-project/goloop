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

package score;

/**
 * Signals a failure of an inter-contract call.
 */
public class RevertedException extends RuntimeException {
    /**
     * Constructs a new exception.
     */
    public RevertedException() {
        super();
    }

    /**
     * Constructs a new exception.
     * @param message message
     */
    public RevertedException(String message) {
        super(message);
    }

    /**
     * Constructs a new exception.
     * @param message message
     * @param cause cause
     */
    public RevertedException(String message, Throwable cause) {
        super(message, cause);
    }

    /**
     * Constructs a new exception.
     * @param cause cause
     */
    public RevertedException(Throwable cause) {
        super(cause);
    }
}
