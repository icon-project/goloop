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
 * Annotation that can be used to indicate whether the method will be exposed externally.
 *
 * <p>In order for a method to be called from outside the contract (EOA or another contract),
 * it needs to be annotated as {@code External}.
 * The annotated methods will be registered on the exportable API list.
 * Any attempt to call a non-external method from outside the contract will fail.
 *
 * <p>If the {@code readonly} element is specified and its value is {@code true}, i.e., {@code @External(readonly=true)},
 * the method will have read-only access to the state DB.
 *
 * <p>NOTE: The special method, named {@code fallback}, cannot be annotated with {@code @External}.
 * (i.e., {@code fallback} method cannot be specified in the transaction message as a callee method,
 * and it can only be called via plain ICX transfer message.)
 */
@Target(ElementType.METHOD)
public @interface External {
    /**
     * The method will have read-only access to the state DB if this value is {@code true}.
     */
    boolean readonly() default false;
}
