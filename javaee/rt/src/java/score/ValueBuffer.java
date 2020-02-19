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

package score;


import java.math.BigInteger;

public class ValueBuffer implements Value {
    public ValueBuffer() {
    }

    public ValueBuffer(byte v) {
    }

    public ValueBuffer(short v) {
    }

    public ValueBuffer(int v) {
    }

    public ValueBuffer(long v) {
    }

    public ValueBuffer(float v) {
    }

    public ValueBuffer(double v) {
    }

    public ValueBuffer(char v) {
    }

    public ValueBuffer(boolean v) {
    }

    public ValueBuffer(BigInteger v) {
    }

    public ValueBuffer(Address v) {
    }

    public ValueBuffer(String v) {
    }

    public ValueBuffer(byte[] v) {
    }

    public ValueBuffer set(byte v) {
        return null;
    }

    public byte asByte() {
        return (byte) 0;
    }

    public ValueBuffer set(short v) {
        return null;
    }

    public short asShort() {
        return (short) 0;
    }

    public ValueBuffer set(int v) {
        return null;
    }

    public int asInt() {
        return 0;
    }

    public ValueBuffer set(long v) {
        return null;
    }

    public long asLong() {
        return 0;
    }

    public ValueBuffer set(float v) {
        return null;
    }

    public float asFloat() {
        return 0;
    }

    public ValueBuffer set(double v) {
        return null;
    }

    public double asDouble() {
        return 0;
    }

    public ValueBuffer set(char v) {
        return null;
    }

    public char asChar() {
        return 0;
    }

    public ValueBuffer set(boolean v) {
        return null;
    }

    public boolean asBoolean() {
        return false;
    }

    public ValueBuffer set(BigInteger v) {
        return null;
    }

    public BigInteger asBigInteger() {
        return null;
    }

    public ValueBuffer set(Address v) {
        return null;
    }

    public Address asAddress() {
        return null;
    }

    public ValueBuffer set(String v) {
        return null;
    }

    public String asString() {
        return null;
    }

    public ValueBuffer set(byte[] v) {
        return null;
    }

    public byte[] asByteArray() {
        return null;
    }

    // TODO implement hashCode and equals
}
