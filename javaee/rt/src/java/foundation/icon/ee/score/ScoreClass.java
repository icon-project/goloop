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

package foundation.icon.ee.score;

import foundation.icon.ee.util.Unshadower;
import i.RuntimeAssertionError;
import org.objectweb.asm.Type;

import java.lang.reflect.Constructor;
import java.lang.reflect.Executable;
import java.lang.reflect.Method;

public class ScoreClass {
    private static final String METHOD_PREFIX = "avm_";

    private final Class<?> cls;

    public ScoreClass(Class<?> cls) {
        this.cls = cls;
    }

    private static boolean hasSameParameterTypes(
            foundation.icon.ee.types.Method em,
            Executable e) {
        var paramClasses = e.getParameterTypes();
        var in = em.getInputs();
        if (paramClasses.length != in.length) {
            return false;
        }
        for (int i=0; i<in.length; ++i) {
            var t = Type.getType(paramClasses[i]);
            var d = Unshadower.unshadowDescriptor(t.getDescriptor());
            if (!d.equals(in[i].getDescriptor())) {
                return false;
            }
        }
        return true;
    }

    private static boolean hasSameReturnType(
            foundation.icon.ee.types.Method em,
            Method m) {
        var t = Type.getType(m.getReturnType());
        var d = Unshadower.unshadowDescriptor(t.getDescriptor());
        return d.equals(em.getOutputDescriptor());
    }

    public Constructor<?> findConstructor(foundation.icon.ee.types.Method em) {
        try {
            var ctors = cls.getConstructors();
            for (var c : ctors) {
                if (hasSameParameterTypes(em, c)) {
                    return c;
                }
            }
        } catch (SecurityException e) {
            RuntimeAssertionError.unexpected(e);
        }
        return null;
    }

    public Method findMethod(foundation.icon.ee.types.Method em) {
        try {
            var cur = cls;
            while (cur != null) {
                var methods = cur.getDeclaredMethods();
                for (var m : methods) {
                    if (m.getName().equals(METHOD_PREFIX + em.getName())
                            && hasSameParameterTypes(em, m)
                            && hasSameReturnType(em, m)) {
                        return m;
                    }
                }
                cur = cur.getSuperclass();
            }
        } catch (SecurityException e) {
            RuntimeAssertionError.unexpected(e);
        }
        return null;
    }
}
