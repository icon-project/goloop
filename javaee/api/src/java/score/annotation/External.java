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
 * </p>
 *
 * <h1> <a id="parameter">Parameter Type</a> </h1>
 * <p>A parameter type of an external method is
 *     <ul>
 *         <li>a <a href="#simple-value">simple value type</a>,</li>
 *         <li>a <a href="#writable-struct">writable struct</a>, or</li>
 *         <li>an array of a <a href="#parameter">parameter type</a></li>
 *     </ul>
 *
 * <h2> <a id="simple-value">Simple Value Type</a> </h2>
 * <p>A simple value type is
 * <ul>
 *     <li>A {@code boolean}, {@code char}, {@code byte},
 *     {@code short}, {@code int} or {@code long}, or</li>
 *     <li> A {@code byte[]}, A {@link java.math.BigInteger},
 *     {@link score.Address} or {@link String}</li>
 * </ul>
 *
 * <h2> <a id="writable-struct">Writable Struct</a> </h2>
 * <p>A type is writable struct if
 * <ul>
 *     <li> the type is a non-abstract class,</li>
 *     <li>
 *         the type has no constructor or public zero argument constructor and
 *     </li>
 *     <li>
 *         the type has a public non-static setter method of
 *         <a href="#writable-property">writable property type</a>.
 *     </li>
 * </ul>
 *
 * <p>For example, the following type is a writable struct.
 * <blockquote><pre>
 *      class Person {
 *          public Person() {...}
 *          public String getName() {...}
 *          public void setName(String name) {...}
 *          public int getAge() {...}
 *          public void setAge(int age) {...}
 *      }
 * </pre></blockquote>
 *
 * <h2> <a id="writable-property">Writable Property</a> </h2>
 * A type is writable property type if the type is
 * <ul>
 *     <li>A <a href="#simple-value">simple value type</a>,</li>
 *     <li>
 *         A {@link Boolean}, {@link Character}, {@link Byte}, {@link Short},
 *         {@link Integer} or {@link Long},
 *     </li>
 *     <li>
 *         A <a href="#writable-struct">writable struct type</a>
 *         (recursion is not allowed), or
 *     </li>
 *     <li>
 *         An array of <a href="#writable-property">writable property</a>
 *     </li>
 * </ul>
 *
 * <h1> <a id="return">Return Type</a> </h1>
 * <p>
 *     A return type of an external method is
 *     <ul>
 *         <li>void,</li>
 *         <li>A <a href="#simple-value">simple value type</a>,</li>
 *         <li>
 *             A <a href="#readable-struct">readable struct</a>
 *             (regarded as a map),
 *         </li>
 *         <li>
 *             an array of a non-void <a href="#return">return type</a>
 *             (regarded as a list),
 *         </li>
 *         <li>
 *             {@link java.util.List} where each element is of a non-void
 *             <a href="#return">return type</a> or null, or
 *         </li>
 *         <li>
 *             {@link java.util.Map} where each key is of a {@code String}
 *             and each value is of a non-void <a href="#return">return type</a>
 *             or null.
 *         </li>
 *     </ul>
 *
 * <h2> <a id="readable-struct">Readable Struct</a> </h2>
 * <p>
 * A type is readable struct if the type has a public non-static getter
 * method of <a href="readable-property">readable property type</a>.
 * For example, the following type is a readable struct.
 * <blockquote><pre>
 *      class Person {
 *          public String getName() {...}
 *          public int getAge() {...}
 *      }
 * </pre></blockquote>
 *
 * <h2> <a id="readable-property">Readable Property</a> </h2>
 * A type is readable property type if the type is
 * <ul>
 *     <li>A <a href="#simple-value">simple value type</a></li>
 *     <li>
 *         A {@link Boolean}, {@link Character}, {@link Byte},
 *        {@link Short}, {@link Integer} and {@link Long},
 *     </li>
 *     <li>
 *         A <a href="#readable-struct">readable struct type</a>
 *         (recursion is not allowed in this case), or
 *     </li>
 *     <li>
 *         An array of <a href="#readable-property">readable property</a>
 *     </li>
 * </ul>
 *
 * <h1>Exception</h1>
 * If an external method is called by an external transaction or an internal
 * transaction and the method throws an exception, the transaction fails.
 * If the exception is an instance of {@link score.UserRevertException} or its
 * subclass, the failure code is decided by the exception. Refer to
 * {@link score.UserRevertException} for details. Otherwise, system code
 * {@code UnknownFailure(1)} is used as the failure code.
 *
 * @see Keep
 * @see score.UserRevertException
 */
@Target(ElementType.METHOD)
public @interface External {
    /**
     * The method will have read-only access to the state DB if this value is {@code true}.
     * @return {@code true} if read only
     */
    boolean readonly() default false;
}
