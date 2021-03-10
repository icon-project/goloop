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

package s.score;

import s.java.lang.RuntimeException;
import s.java.lang.String;
import s.java.lang.Throwable;

public class RevertedException extends RuntimeException {
    public RevertedException() {
        super();
    }

    public RevertedException(String message) {
        super(message);
    }

    public RevertedException(String message, Throwable cause) {
        super(message, cause);
    }

    public RevertedException(Throwable cause) {
        super(cause);
    }
}
