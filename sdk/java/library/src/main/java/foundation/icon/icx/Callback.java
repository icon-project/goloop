/*
 * Copyright 2018 ICON Foundation
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

package foundation.icon.icx;

/**
 *  A callback of the asynchronous execution
 */
public interface Callback<T> {

    /**
     * Invoked when the execution is successful
     * @param result a result of the execution
     */
    void onSuccess(T result);

    /**
     * Invoked when the execution is completed with an exception
     * @param exception an exception thrown during the execution
     */
    void onFailure(Exception exception);
}
