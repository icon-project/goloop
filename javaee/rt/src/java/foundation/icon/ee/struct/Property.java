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

package foundation.icon.ee.struct;

import java.lang.reflect.Method;
import java.lang.reflect.Modifier;
import java.util.ArrayList;
import java.util.Arrays;
import java.util.List;

import static i.PackageConstants.kUserDotPrefix;

public interface Property {
    int METHOD_PREFIX_LEN = 7;

    static String decapitalize(String s) {
        if (s.length() > 1 && Character.isUpperCase(s.charAt(0))
                && Character.isUpperCase(s.charAt(1))) {
            return s;
        }
        return Character.toLowerCase(s.charAt(0)) + s.substring(1);
    }

    static String capitalize(String s) {
        if (s.length() > 1 && Character.isLowerCase(s.charAt(0))
                && Character.isUpperCase(s.charAt(1))) {
            return s;
        }
        return Character.toUpperCase(s.charAt(0)) + s.substring(1);
    }

    static WritableProperty getWritableProperty(
            Class<?> cls,
            String property) {
        var setter = "avm_set" + capitalize(property);
        while (cls != null && cls.getName().startsWith(kUserDotPrefix)) {
            var ma = Arrays.stream(cls.getDeclaredMethods()).filter(
                    m -> m.getName().equals(setter)
                            && m.getParameterCount() == 1
                            && m.getReturnType() == void.class
                            && !Modifier.isStatic(m.getModifiers())
                            && Modifier.isPublic(m.getModifiers())
            ).toArray(Method[]::new);
            if (ma.length == 1) {
                return new WritableMethodProperty(ma[0]);
            }
            try {
                var f = cls.getDeclaredField(property);
                if (!Modifier.isStatic(f.getModifiers())
                        && Modifier.isPublic(f.getModifiers())) {
                    return new FieldProperty(f);
                }
            } catch (NoSuchFieldException ignored) {
            }
            cls = cls.getSuperclass();
        }
        return null;
    }

    static List<ReadableProperty> getReadableProperties(Object obj) {
        var cls = obj.getClass();
        final var prefix = "avm_get";
        var props = new ArrayList<ReadableProperty>();
        while (cls != null && cls.getName().startsWith(kUserDotPrefix)) {
            Arrays.stream(cls.getDeclaredMethods())
                    .filter(m -> m.getName().startsWith(prefix)
                            && m.getParameterCount() == 0
                            && m.getReturnType() != void.class
                            && !Modifier.isStatic(m.getModifiers())
                            && Modifier.isPublic(m.getModifiers())
                    )
                    .map(ReadableMethodProperty::new)
                    .forEachOrdered(props::add);
            Arrays.stream(cls.getDeclaredFields())
                    .filter(f -> (!Modifier.isStatic(f.getModifiers())
                            && Modifier.isPublic(f.getModifiers()))
                    )
                    .filter(f -> {
                        for (var p : props) {
                            if (f.getName().equals(p.getName())) {
                                return true;
                            }
                        }
                        return false;
                    })
                    .map(FieldProperty::new)
                    .forEachOrdered(props::add);
            cls = cls.getSuperclass();
        }
        return props;
    }

    String getName();
    Class<?> getType();
}
