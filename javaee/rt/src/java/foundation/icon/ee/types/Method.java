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

import org.objectweb.asm.commons.Remapper;

import java.util.Arrays;
import java.util.Objects;

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
        public static final int MAX_FLAG = 8;
    }

    public static class DataType {
        public static final int NONE = 0;
        public static final int INTEGER = 1;
        public static final int STRING = 2;
        public static final int BYTES = 3;
        public static final int BOOL = 4;
        public static final int ADDRESS = 5;
        public static final int LIST = 6;
        public static final int DICT = 7;
        public static final int STRUCT = 8;

        public static final int DIMENSION_SHIFT = 4;
        public static final int ELEMENT_MASK = (1<< DIMENSION_SHIFT)-1;

        public static int getElement(int type) {
            return type&ELEMENT_MASK;
        }
    }

    public static class TypeDetail {
        int type;
        Field[] structFields; // non-null iff type&STRUCT

        public TypeDetail(int type) {
            this.type = type;
        }

        public TypeDetail(int type, Field[] structFields) {
            this.type = type;
            this.structFields = structFields;
        }

        public int getType() {
            return type;
        }

        public Field[] getStructFields() {
            return structFields;
        }

        @Override
        public String toString() {
            return "TypeDetail{" +
                    "type=" + type +
                    (structFields!=null
                            ? ", structFields=" + Arrays.toString(structFields)
                            : "") +
                    '}';
        }

        @Override
        public boolean equals(Object o) {
            if (this == o) return true;
            if (o == null || getClass() != o.getClass()) return false;
            TypeDetail that = (TypeDetail) o;
            return type == that.type &&
                    Arrays.equals(structFields, that.structFields);
        }

        @Override
        public int hashCode() {
            int result = Objects.hash(type);
            result = 31 * result + Arrays.hashCode(structFields);
            return result;
        }
    }

    public static class Field {
        String name;
        TypeDetail typeDetail;

        public Field(String name, TypeDetail typeDetail) {
            this.name = name;
            this.typeDetail = typeDetail;
        }

        public Field(String name, int type, Field[] structTypes) {
            this.name = name;
            this.typeDetail = new TypeDetail(type, structTypes);
        }

        public String getName() {
            return name;
        }

        public TypeDetail getTypeDetail() {
            return typeDetail;
        }

        public int getType() {
            return typeDetail.getType();
        }

        public Field[] getStructFields() {
            return typeDetail.getStructFields();
        }

        @Override
        public String toString() {
            return "Field{" +
                    "name='" + name + '\'' +
                    ", type=" + typeDetail.type +
                    (typeDetail.structFields!=null
                            ? ", structFields=" + Arrays.toString(typeDetail.structFields)
                            : "") +
                    '}';
        }

        @Override
        public boolean equals(Object o) {
            if (this == o) return true;
            if (o == null || getClass() != o.getClass()) return false;
            Field field = (Field) o;
            return name.equals(field.name) &&
                    typeDetail.equals(field.typeDetail);
        }

        @Override
        public int hashCode() {
            return Objects.hash(name, typeDetail);
        }
    }

    public static class Parameter {
        String name;
        String descriptor;
        TypeDetail typeDetail;
        boolean optional;

        public Parameter(String name, String descriptor, int type) {
            this(name, descriptor, new TypeDetail(type), false);
        }

        public Parameter(String name, String descriptor, int type,
                boolean optional) {
            this(name, descriptor, new TypeDetail(type), optional);
        }

        public Parameter(String name, String descriptor, int type,
                Field[] structFields, boolean optional) {
            this(name, descriptor, new TypeDetail(type, structFields),
                    optional);
        }

        public Parameter(String name, String descriptor, TypeDetail typeDetail, boolean optional) {
            this.name = name;
            this.descriptor = descriptor;
            this.typeDetail = typeDetail;
            this.optional = optional;
        }

        public Parameter remap(Remapper remapper) {
            return new Parameter(
                    name,
                    remapper.mapDesc(descriptor),
                    typeDetail,
                    optional
            );
        }

        public String getName() {
            return name;
        }

        public String getDescriptor() {
            return descriptor;
        }

        public int getType() {
            return typeDetail.getType();
        }

        public boolean isOptional() {
            return optional;
        }

        public Field[] getStructFields() {
            return typeDetail.getStructFields();
        }

        public TypeDetail getTypeDetail() {
            return typeDetail;
        }

        @Override
        public String toString() {
            return "Parameter{" +
                    "name='" + name + '\'' +
                    (descriptor.isEmpty() ? "" : ", descriptor=" + descriptor) +
                    ", type=" + typeDetail.type +
                    (typeDetail.structFields!=null
                            ? ", structFields=" + Arrays.toString(typeDetail.structFields)
                            : "") +
                    ", optional=" + optional +
                    '}';
        }

        @Override
        public boolean equals(Object o) {
            if (this == o) return true;
            if (o == null || getClass() != o.getClass()) return false;
            Parameter parameter = (Parameter) o;
            return optional == parameter.optional &&
                    name.equals(parameter.name) &&
                    descriptor.equals(parameter.descriptor) &&
                    typeDetail.equals(parameter.typeDetail);
        }

        @Override
        public int hashCode() {
            return Objects.hash(name, descriptor, typeDetail, optional);
        }
    }

    private final int type;
    private final String name;
    private final int flags;
    private final int indexed;
    private final Parameter[] inputs;
    private final int output;
    private final String outputDescriptor;

    private Method(int type, String name, int flags, int indexed, Parameter[] inputs, int output, String outputDescriptor) {
        this.type = type;
        this.name = name;
        this.flags = flags;
        this.indexed = indexed;
        this.inputs = inputs;
        this.output = output;
        this.outputDescriptor = outputDescriptor;
    }

    public Method remap(Remapper remapper) {
        return new Method(
                type,
                name,
                flags,
                indexed,
                Arrays.stream(inputs)
                        .map(in -> in.remap(remapper))
                        .toArray(Parameter[]::new),
                output,
                remapper.mapDesc(outputDescriptor)
        );
    }

    public static Method newFunction(String name, int flags, Parameter[] inputs, int output, String outputDescriptor) {
        return new Method(MethodType.FUNCTION, name, flags,
                (inputs != null) ? inputs.length : 0, inputs, output,
                outputDescriptor);
    }

    public static Method newFunction(String name, int flags, int optional, Parameter[] inputs, int output, String outputDescriptor) {
        if (optional > 0) {
            return new Method(MethodType.FUNCTION, name, flags,
                    inputs.length - optional, inputs, output, outputDescriptor);
        }
        return newFunction(name, flags, inputs, output, outputDescriptor);
    }

    public static Method newFallback() {
        return new Method(MethodType.FALLBACK, "fallback", Flags.PAYABLE, 0, new Parameter[0], 0, "V");
    }

    public static Method newEvent(String name, int indexed, Parameter[] inputs) {
        return new Method(MethodType.EVENT, name, 0, indexed, inputs, 0, "V");
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
                (outputDescriptor.isEmpty() ? "" : ", outputDescriptor=" + outputDescriptor) +
                '}';
    }

    public String getOutputDescriptor() {
        return outputDescriptor;
    }

    public String getDescriptor() {
        var sb = new StringBuilder();
        sb.append('(');
        for (var p : inputs) {
            sb.append(p.getDescriptor());
        }
        sb.append(')');
        sb.append(getOutputDescriptor());
        return sb.toString();
    }

    public String getDebugName() {
        if (outputDescriptor.isEmpty()) {
            return name;
        }
        return name + getDescriptor();
    }

    @Override
    public boolean equals(Object o) {
        if (this == o) return true;
        if (o == null || getClass() != o.getClass()) return false;
        Method method = (Method) o;
        return type == method.type &&
                flags == method.flags &&
                indexed == method.indexed &&
                output == method.output &&
                name.equals(method.name) &&
                Arrays.equals(inputs, method.inputs) &&
                outputDescriptor.equals(method.outputDescriptor);
    }

    @Override
    public int hashCode() {
        int result = Objects.hash(type, name, flags, indexed, output, outputDescriptor);
        result = 31 * result + Arrays.hashCode(inputs);
        return result;
    }
}
