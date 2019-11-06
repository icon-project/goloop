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

package foundation.icon.ee.types;

import java.math.BigInteger;
import java.util.Arrays;
import java.util.Map;

public class Method {

    public static class MethodType {
        public static final int FUNCTION = 0;
        public static final int FALLBACK = 1;
        public static final int EVENT = 2;
    }

    public static class Flags {
        public static final int READONLY = 1;
        public static final int EXTERNAL = 2;
        public static final int PAYABLE = 4;
        public static final int ISOLATED = 8;
    }

    public static class DataType {
        public static final int NONE = 0;
        public static final int INTEGER = 1;
        public static final int STRING = 2;
        public static final int BYTES = 3;
        public static final int BOOL = 4;
        public static final int ADDRESS = 5;
    }

    public static class Parameter {
        String name;
        String descriptor;
        int type;
        boolean optional;

        public Parameter(String name, String descriptor, int type) {
            this.name = name;
            this.descriptor = descriptor;
            this.type = type;
        }

        public Parameter(String name, String descriptor, int type, boolean optional) {
            this.name = name;
            this.descriptor = descriptor;
            this.type = type;
            this.optional = optional;
        }

        public String getName() {
            return name;
        }

        public String getDescriptor() {
            return descriptor;
        }

        public int getType() {
            return type;
        }

        public boolean isOptional() {
            return optional;
        }

        @Override
        public String toString() {
            return "Parameter{" +
                    "name='" + name + '\'' +
                    ", descriptor=" + descriptor +
                    ", type=" + type +
                    ", optional=" + optional +
                    '}';
        }
    }

    private int type;
    private String name;
    private int flags;
    private int indexed;
    private Parameter[] inputs;
    private int output;

    private Method(int type, String name, int flags, int indexed, Parameter[] inputs, int output) {
        this.type = type;
        this.name = name;
        this.flags = flags;
        this.indexed = indexed;
        this.inputs = inputs;
        this.output = output;
    }

    public static Method newFunction(String name, int flags, Parameter[] inputs, int output) {
        return new Method(MethodType.FUNCTION, name, flags,
                (inputs != null) ? inputs.length : 0, inputs, output);
    }

    public static Method newFunction(String name, int flags, int optional, Parameter[] inputs, int output) {
        if (optional > 0) {
            return new Method(MethodType.FUNCTION, name, flags,
                    inputs.length - optional, inputs, output);
        }
        return newFunction(name, flags, inputs, output);
    }

    public static Method newFallback() {
        return new Method(MethodType.FALLBACK, "fallback", Flags.PAYABLE, 0, new Parameter[0], 0);
    }

    public static Method newEvent(String name, int indexed, Parameter[] inputs) {
        return new Method(MethodType.EVENT, name, 0, indexed, inputs, 0);
    }

    public int getType() {
        return type;
    }

    public String getName() {
        return name;
    }

    public int getFlags() {
        return flags;
    }

    public int getIndexed() {
        return indexed;
    }

    public Parameter[] getInputs() {
        return inputs;
    }

    public int getOutput() {
        return output;
    }

    @Override
    public String toString() {
        return "Method{" +
                "type=" + type +
                ", name='" + name + '\'' +
                ", flags=" + flags +
                ", indexed=" + indexed +
                ", inputs=" + Arrays.toString(inputs) +
                ", output=" + output +
                '}';
    }

    private static final Map<String, Class<?>> descToClass = Map.of(
            "Z", boolean.class,
            "C", char.class,
            "B", byte.class,
            "S", short.class,
            "I", int.class,
            "J", long.class,
            "Ljava/math/BigInteger;", s.java.math.BigInteger.class,
            "Ljava/lang/String;", s.java.lang.String.class,
            "[B", a.ByteArray.class,
            "Lavm/Address;", p.avm.Address.class
    );

    public boolean hasValidParams() {
        for (Parameter p: inputs) {
            if (!descToClass.containsKey(p.getDescriptor()))
                return false;
        }
        return true;
    }

    public Class<?>[] getParameterClasses() {
        Class<?>[] out = new Class<?>[inputs.length];
        for (int i=0; i<inputs.length; i++) {
            out[i] = descToClass.get(inputs[i].getDescriptor());
        }
        return out;
    }

    public s.java.lang.Object[] convertParameters(Object[] params) {
        assert params.length == inputs.length : String.format("bad param length=%d input length=%d", params.length, inputs.length);

        s.java.lang.Object[] out = new s.java.lang.Object[inputs.length];
        for (int i=0; i<inputs.length; i++) {
            var d = inputs[i].getDescriptor();
            if (d.equals("Z")) {
                Boolean p = (Boolean) params[i];
                out[i] = s.java.lang.Boolean.valueOf(p.booleanValue());
            } else if (d.equals("C")) {
                BigInteger p = (BigInteger) params[i];
                out[i] = s.java.lang.Character.valueOf((char)p.intValue());
            } else if (d.equals("B")) {
                BigInteger p = (BigInteger) params[i];
                out[i] = s.java.lang.Byte.valueOf(p.byteValue());
            } else if (d.equals("S")) {
                BigInteger p = (BigInteger) params[i];
                out[i] = s.java.lang.Short.valueOf(p.shortValue());
            } else if (d.equals("I")) {
                BigInteger p = (BigInteger) params[i];
                out[i] = s.java.lang.Integer.valueOf(p.intValue());
            } else if (d.equals("J")) {
                BigInteger p = (BigInteger) params[i];
                out[i] = s.java.lang.Long.valueOf(p.longValue());
            } else if (d.equals("Ljava/math/BigInteger;")) {
                BigInteger p = (BigInteger) params[i];
                out[i] = new s.java.math.BigInteger(p);
            } else if (d.equals("Ljava/lang/String;")) {
                String p = (String) params[i];
                out[i] = new s.java.lang.String(p);
            } else if (d.equals("[B")) {
                byte[] p = (byte[]) params[i];
                out[i] = new a.ByteArray(p);
            } else if (d.equals("Lavm/Address;")) {
                Address p = (Address) params[i];
                out[i] = new p.avm.Address(p.toByteArray());
            } else {
                assert false : String.format("bad %d-th param type %s", i, params[i].getClass().getName());
            }
        }
        return out;
    }
}
