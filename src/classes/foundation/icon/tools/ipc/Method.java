/*
 * Copyright (c) 2019 ICON Foundation
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

package foundation.icon.tools.ipc;

import org.msgpack.core.MessageBufferPacker;

import java.io.IOException;

class Method {

    class MethodType {
        static final int FUNCTION = 0;
        static final int FALLBACK = 1;
        static final int EVENT = 2;
    }

    class Flags {
        static final int READONLY = 1;
        static final int EXTERNAL = 2;
        static final int PAYABLE = 4;
        static final int ISOLATED = 8;
    }

    class DataType {
        static final int NONE = 0;
        static final int INTEGER = 1;
        static final int STRING = 2;
        static final int BYTES = 3;
        static final int BOOL = 4;
        static final int ADDRESS = 5;
    }

    static class Parameter {
        String name;
        int type;

        Parameter(String name, int type) {
            this.name = name;
            this.type = type;
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

    static Method newFunction(String name, int flags, Parameter[] inputs, int output) {
        return new Method(MethodType.FUNCTION, name, flags,
                (inputs != null) ? inputs.length : 0, inputs, output);
    }

    static Method newFallback() {
        return new Method(MethodType.FALLBACK, "fallback", Flags.PAYABLE, 0, null, 0);
    }

    static Method newEvent(String name, int indexed, Parameter[] inputs) {
        return new Method(MethodType.EVENT, name, 0, indexed, inputs, 0);
    }

    void accept(MessageBufferPacker packer) throws IOException {
        packer.packArrayHeader(6);
        packer.packInt(type);
        packer.packString(name);
        packer.packInt(flags);
        packer.packInt(indexed);
        if (inputs != null) {
            packer.packArrayHeader(inputs.length);
            for (Parameter p : inputs) {
                packer.packArrayHeader(3);
                packer.packString(p.name);
                packer.packInt(p.type);
                packer.packNil();
            }
        } else {
            packer.packArrayHeader(0);
        }
        if (output != 0) {
            packer.packArrayHeader(1);
            packer.packInt(output);
        } else {
            packer.packArrayHeader(0);
        }
    }
}
