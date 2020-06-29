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
 * Annotation that can be used to indicate whether the method can receive ICX coins.
 *
 * <p>If this annotation is applied to the external method, the method can receive the incoming ICX coins
 * designated in the transaction message and process further works for it.
 * Users can get the value of transferred ICX coins by using {@link score.Context#getValue()}.
 *
 * <p>If ICX coins were passed to a non-payable method, that transaction would fail.
 *
 * <p>NOTE: The special method, named {@code fallback}, is invoked whenever the contract receives
 * plain ICX coins without data.  However, if the {@code fallback} method was not annotated with {@code @Payable},
 * it would not be listed on the SCORE APIs and could not be called as well.
 */
@Target(ElementType.METHOD)
public @interface Payable {
}
