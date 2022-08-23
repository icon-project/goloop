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
 * Annotation that can be used to indicate whether the method parameter is optional.
 *
 * <p>If a method parameter is annotated with this {@code Optional}, the parameter can be omitted
 * in the transaction message.
 * If optional parameters were omitted when the external method is called, the value of optional parameters
 * would be their zero values.
 * The zero value is:
 *     0 for numeric types (including BigInteger),
 *     false for the boolean type, and
 *     null for Object types.
 */
@Target(ElementType.PARAMETER)
public @interface Optional {
}
