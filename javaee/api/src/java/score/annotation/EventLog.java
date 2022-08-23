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

package score.annotation;

import java.lang.annotation.ElementType;
import java.lang.annotation.Target;

/**
 * Annotation that can be used to record logs in its TxResult as {@code eventLogs}.
 *
 * <p> If the value of an element, named {@code indexed}, is set, the designated number of parameters
 * of the applied method declaration will be indexed in the order and included in the Bloom filter.
 * Indexed parameters and non-indexed parameters are separately stored in the TxResult.
 * At most 3 parameters can be indexed, and the value of {@code indexed} cannot exceed the number of parameters.
 * Possible data types for method parameters are {@code int}, {@code boolean}, {@code byte[]},
 * {@code BigInteger}, {@code String}, and {@code Address}.
 *
 * <p>It is recommended to declare a method without an implementation body.
 * Even if the applied method has the body, it does not be executed in runtime.
 */
@Target(ElementType.METHOD)
public @interface EventLog {
    /**
     * The number of indexed parameters of the applied method declaration (maximum 3).
     */
    int indexed() default 0;
}
