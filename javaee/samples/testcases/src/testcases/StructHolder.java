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

package testcases;

import score.Address;
import score.annotation.External;

import java.math.BigInteger;

public class StructHolder {
    public static class SimpleStruct {
        private String string;
        private BigInteger integer;
        private Address address;
        private boolean bool;
        private byte[] bytes;

        public SimpleStruct() {
        }

        public String getString() {
            return string;
        }

        public void setString(String string) {
            this.string = string;
        }

        public BigInteger getInteger() {
            return integer;
        }

        public void setInteger(BigInteger integer) {
            this.integer = integer;
        }

        public Address getAddress() {
            return address;
        }

        public void setAddress(Address address) {
            this.address = address;
        }

        public boolean isBool() {
            return bool;
        }

        public void setBool(boolean bool) {
            this.bool = bool;
        }

        public byte[] getBytes() {
            return bytes;
        }

        public void setBytes(byte[] bytes) {
            this.bytes = bytes;
        }
    }

    public static class ComplexStruct extends SimpleStruct {
        private SimpleStruct simpleStruct;

        public ComplexStruct() {
        }

        public SimpleStruct getSimpleStruct() {
            return simpleStruct;
        }

        public void setSimpleStruct(SimpleStruct simpleStruct) {
            this.simpleStruct = simpleStruct;
        }
    }

    public StructHolder() {
    }

    private SimpleStruct simpleStruct;

    @External(readonly=true)
    public SimpleStruct getSimpleStruct() {
        return simpleStruct;
    }

    @External
    public void setSimpleStruct(SimpleStruct simpleStruct) {
        this.simpleStruct = simpleStruct;
    }

    private ComplexStruct complexStruct;

    @External(readonly=true)
    public ComplexStruct getComplexStruct() {
        return complexStruct;
    }

    @External
    public void setComplexStruct(ComplexStruct complexStruct) {
        this.complexStruct = complexStruct;
    }
}
