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

import java.util.Arrays;

public class Method {

    static class MethodType {
        static final int FUNCTION = 0;
        static final int FALLBACK = 1;
        static final int EVENT = 2;
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
        int type;
        boolean optional;

        public Parameter(String name, int type) {
            this.name = name;
            this.type = type;
        }

        public Parameter(String name, int type, boolean optional) {
            this.name = name;
            this.type = type;
            this.optional = optional;
        }

        public String getName() {
            return name;
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
        return new Method(MethodType.FALLBACK, "fallback", Flags.PAYABLE, 0, null, 0);
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
}
