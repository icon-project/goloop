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

package score.annotation;

import java.lang.annotation.ElementType;
import java.lang.annotation.Target;

/**
 *  Denotes that the element should not be removed when the code is optimized by
 *  the tool kit.
 *
 *  <p>The tool kit removes unused methods during the optimization phase.
 *  If a struct appears in the signature of an {@link External @External}
 *  method, necessary constructor, getters and setters for the struct survive
 *  the optimization phase even if the methods are not accessed in user code.
 *  More specifically, if a class is a writable struct and the class is used as
 *  a parameter type of an external method, its zero argument constructor (if it
 *  is defined) and property setters are not removed.
 *  Also, a class is a readable struct and the class is used as a return type of
 *  an external method, its getters are not removed.
 *  For example, in the following contract, {@code @Keep} annotation is not
 *  necessary for {@code Person} class since it is used as a parameter of
 *  an external method.
 *  </p>
 *  <blockquote><pre>
 *      // Person.java
 *      public class Person {
 *          public Person() {...}
 *          public String getName() {...}
 *          public void setName(String name) {...}
 *      }
 *
 *      // Callee.java
 *      public class Score {
 *          &#64;External
 *          public void hello(Person person) {
 *              Context.println("Hello " + person.getName());
 *          }
 *      }
 *  </pre></blockquote>
 *
 *  <p>There are some cases that a user must manually use {@code @Keep}
 *  annotation.
 *  When a struct is passed as an argument of {@link score.Context#call},
 *  getters are called by system.
 *  Also, when a struct is returned by {@link score.Context#call}, a constructor
 *  and setters are called by system.
 *  However, the tool kit cannot track the runtime type of the parameter of the
 *  {@link score.Context#call} method.
 *  Thus, the getters or setters are removed if they are not accessed in
 *  user code and they are not used as a parameter type or a return type of an
 *  external method.
 *  In that case, {@code @Keep} annotation is required for the necessary methods
 *  not to be optimized away.
 *  For example, in the following contract, {@code @Keep} annotation is
 *  necessary for {@code getName} because user code do not call the method and
 *  the {@code Person} class is not used as a parameter or a return type of an
 *  {@link External @External} method.
 *  </p>
 *  <blockquote><pre>
 *      // Person.java
 *      class Person {
 *          public Person(name String) {...}
 *
 *          &#64;Keep
 *          public String getName() {...}
 *      }
 *
 *      // Caller.java
 *      public class Caller {
 *          &#64;External
 *          public void test(Address addr) {
 *              Context.call(addr, "hello", new Person("Kim"));
 *          }
 *      }
 *  </pre></blockquote>
 *
 *  <p>For the definition of readable struct and writable struct, refer to the
 *  document of {@link External @External}</p>
 *
 *  @see External
 */
@Target({ElementType.METHOD, ElementType.FIELD, ElementType.CONSTRUCTOR})
public @interface Keep {
}
