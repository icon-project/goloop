/*
 * Copyright 2023 ICON Foundation
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

package foundation.icon.ee;

import foundation.icon.ee.io.RLPCodecTest;
import foundation.icon.ee.io.RLPNCodecTest;
import foundation.icon.ee.test.SimpleTest;
import org.junit.jupiter.api.Test;
import score.Address;
import score.ByteArrayObjectWriter;
import score.Context;
import score.ObjectReader;
import score.ObjectWriter;
import score.annotation.External;

import java.math.BigInteger;
import java.util.Arrays;

public class CodecTest4 extends SimpleTest {

    public static class Score {
        public interface Writer<T> {
            void write(ObjectWriter ow, T v);
        }

        public interface Reader<T> {
            T read(ObjectReader or);
        }

        private static final String hexDigits = "0123456789abcdef";

        public static String hex(byte[] ba) {
            var sb = new StringBuilder();
            for (int i = 0; i < ba.length; i++) {
                sb.append(hexDigits.charAt((ba[i] >> 4) & 0xf));
                sb.append(hexDigits.charAt((ba[i]) & 0xf));
            }
            return sb.toString();
        }

        static class RLPAssertion implements RLPCodecTest.Assertion, RLPNCodecTest.Assertion {
            private final String codec;

            public RLPAssertion(String codec) {
                this.codec = codec;
            }

            <T> void assertCodingEqualGeneric(String exp, T v, Writer<T> writer, Reader<T> reader) {
                var ow = Context.newByteArrayObjectWriter(codec);
                writer.write(ow, v);
                var ba = ow.toByteArray();
                Context.require(exp.equals(hex(ba)), "encoding fail in test exp=" + exp + " val=" + v);
                var or = Context.newByteArrayObjectReader(codec, ba);
                Context.require(v.equals(reader.read(or)), "decoding fail in test exp=" + exp + " val=" + v);
            }

            void assertCodingEqualObject(String exp, Object v, Class cls) {
                var ow = Context.newByteArrayObjectWriter(codec);
                ow.write(v);
                var ba = ow.toByteArray();
                Context.require(exp.equals(hex(ba)), "encoding fail in test exp=" + exp + " val=" + v);
                var or = Context.newByteArrayObjectReader(codec, ba);
                Context.require(v.equals(or.read(cls)), "decoding fail in test exp=" + exp + " val=" + v);
            }

            void assertCodingEqualObjects(String exp, Object v, Class cls) {
                var ow = Context.newByteArrayObjectWriter(codec);
                ow.write(v, v);
                var ba = ow.toByteArray();
                Context.require((exp + exp).equals(hex(ba)), "encoding fail in test exp=" + exp + " val=" + v);
                var or = Context.newByteArrayObjectReader(codec, ba);
                Context.require(v.equals(or.read(cls)), "decoding fail in test exp=" + exp + " val=" + v);
                Context.require(v.equals(or.read(cls)), "decoding fail in test exp=" + exp + " val=" + v);
            }

            <T, TH extends Throwable> void assertWriteThrowsGeneric(Class<TH> exp, T v, Writer<T> writer) {
                boolean throwable = false;
                try {
                    var ow = Context.newByteArrayObjectWriter(codec);
                    writer.write(ow, v);
                } catch (Throwable e) {
                    Context.require(e.getClass().getName().equals(exp.getName()), "unexpected exception thrown exp=" + exp.getName() + " actual=" + e.getClass().getName());
                    throwable = true;
                }
                Context.require(throwable, "nothing is thrown");
            }

            public void assertCodingEquals(String exp, boolean v) {
                assertCodingEqualGeneric(
                        exp, v,
                        new Writer<>() {
                            public void write(ObjectWriter ow, Boolean v) {
                                ow.write((boolean) v);
                            }
                        },
                        new Reader<>() {
                            public Boolean read(ObjectReader or) {
                                return or.readBoolean();
                            }
                        }
                );
                assertCodingEqualObject(exp, v, Boolean.class);
                assertCodingEqualObjects(exp, v, Boolean.class);
            }

            private final Writer<Byte> byteWriter = new Writer<>() {
                public void write(ObjectWriter ow, Byte v) {
                    ow.write((byte) v);
                }
            };

            private final Reader<Byte> byteReader = new Reader<>() {
                public Byte read(ObjectReader or) {
                    return or.readByte();
                }
            };

            public void assertCodingEquals(String exp, byte v) {
                assertCodingEqualGeneric(
                        exp, v, byteWriter, byteReader
                );
                assertCodingEqualObject(exp, v, Byte.class);
                assertCodingEqualObjects(exp, v, Byte.class);
            }

            @Override
            public <TH extends Throwable> void assertWriteThrows(Class<TH> exp, byte v) {
                assertWriteThrowsGeneric(exp, v, byteWriter);
            }

            private final Writer<Short> shortWriter = new Writer<>() {
                public void write(ObjectWriter ow, Short v) {
                    ow.write((short) v);
                }
            };

            private final Reader<Short> shortReader = new Reader<>() {
                public Short read(ObjectReader or) {
                    return or.readShort();
                }
            };

            public void assertCodingEquals(String exp, short v) {
                assertCodingEqualGeneric(
                        exp, v, shortWriter, shortReader
                );
                assertCodingEqualObject(exp, v, Short.class);
                assertCodingEqualObjects(exp, v, Short.class);
            }

            @Override
            public <TH extends Throwable> void assertWriteThrows(Class<TH> exp, short v) {
                assertWriteThrowsGeneric(exp, v, shortWriter);
            }

            public void assertCodingEquals(String exp, char v) {
                assertCodingEqualGeneric(
                        exp, v,
                        new Writer<>() {
                            public void write(ObjectWriter ow, Character v) {
                                ow.write((char) v);
                            }
                        },
                        new Reader<>() {
                            public Character read(ObjectReader or) {
                                return or.readChar();
                            }
                        }
                );
                assertCodingEqualObject(exp, v, Character.class);
                assertCodingEqualObjects(exp, v, Character.class);
            }

            private final Writer<Integer> intWriter = new Writer<>() {
                public void write(ObjectWriter ow, Integer v) {
                    ow.write((int) v);
                }
            };

            private final Reader<Integer> intReader = new Reader<>() {
                public Integer read(ObjectReader or) {
                    return or.readInt();
                }
            };

            public void assertCodingEquals(String exp, int v) {
                assertCodingEqualGeneric(
                        exp, v, intWriter, intReader
                );
                assertCodingEqualObject(exp, v, Integer.class);
                assertCodingEqualObjects(exp, v, Integer.class);
            }

            @Override
            public <TH extends Throwable> void assertWriteThrows(Class<TH> exp, int v) {
                assertWriteThrowsGeneric(exp, v, intWriter);
            }

            public void assertCodingEquals(String exp, float v) {
                assertCodingEqualGeneric(
                        exp, v,
                        new Writer<>() {
                            public void write(ObjectWriter ow, Float v) {
                                ow.write((float) v);
                            }
                        },
                        new Reader<>() {
                            public Float read(ObjectReader or) {
                                return or.readFloat();
                            }
                        }
                );
                assertCodingEqualObject(exp, v, Float.class);
                assertCodingEqualObjects(exp, v, Float.class);
            }

            private final Writer<Long> longWriter = new Writer<>() {
                public void write(ObjectWriter ow, Long v) {
                    ow.write((long) v);
                }
            };

            private final Reader<Long> longReader = new Reader<>() {
                public Long read(ObjectReader or) {
                    return or.readLong();
                }
            };

            public void assertCodingEquals(String exp, long v) {
                assertCodingEqualGeneric(
                        exp, v, longWriter, longReader
                );
                assertCodingEqualObject(exp, v, Long.class);
                assertCodingEqualObjects(exp, v, Long.class);
            }

            @Override
            public <TH extends Throwable> void assertWriteThrows(Class<TH> exp, long v) {
                assertWriteThrowsGeneric(exp, v, longWriter);
            }

            public void assertCodingEquals(String exp, double v) {
                assertCodingEqualGeneric(
                        exp, v,
                        new Writer<>() {
                            public void write(ObjectWriter ow, Double v) {
                                ow.write((double) v);
                            }
                        },
                        new Reader<>() {
                            public Double read(ObjectReader or) {
                                return or.readDouble();
                            }
                        }
                );
                assertCodingEqualObject(exp, v, Double.class);
                assertCodingEqualObjects(exp, v, Double.class);
            }

            private final Writer<BigInteger> biWriter = new Writer<>() {
                public void write(ObjectWriter ow, BigInteger v) {
                    ow.write(v);
                }
            };

            private final Reader<BigInteger> biReader = new Reader<>() {
                public BigInteger read(ObjectReader or) {
                    return or.readBigInteger();
                }
            };

            @Override
            public void assertCodingEquals(String exp, BigInteger v) {
                assertCodingEqualGeneric(exp, v, biWriter, biReader);
                assertCodingEqualObject(exp, v, BigInteger.class);
                assertCodingEqualObjects(exp, v, BigInteger.class);
            }

            @Override
            public <TH extends Throwable> void assertWriteThrows(Class<TH> exp, BigInteger v) {
                assertWriteThrowsGeneric(exp, v, biWriter);
            }

            @Override
            public void assertCodingEquals(String exp, String v) {
                assertCodingEqualGeneric(exp, v,
                        new Writer<>() {
                            public void write(ObjectWriter ow, String v) {
                                ow.write(v);
                            }
                        },
                        new Reader<>() {
                            public String read(ObjectReader or) {
                                return or.readString();
                            }
                        }
                );
                assertCodingEqualObject(exp, v, String.class);
                assertCodingEqualObjects(exp, v, String.class);
            }

            @Override
            public void assertCodingEquals(String exp, byte[] v) {
                do {
                    var ow = Context.newByteArrayObjectWriter(codec);
                    ow.write(v);
                    var ba = ow.toByteArray();
                    Context.require(exp.equals(hex(ba)), "encoding fail in test exp=" + exp + " val=" + v);
                    var or = Context.newByteArrayObjectReader(codec, ba);
                    Context.require(Arrays.equals(v, or.readByteArray()), "decoding fail in test exp=" + exp + " val=" + v);
                } while (false);
                do {
                    var ow = Context.newByteArrayObjectWriter(codec);
                    ow.write((Object) v);
                    var ba = ow.toByteArray();
                    Context.require(exp.equals(hex(ba)), "encoding fail in test exp=" + exp + " val=" + v);
                    var or = Context.newByteArrayObjectReader(codec, ba);
                    Context.require(Arrays.equals(v, or.read(byte[].class)), "decoding fail in test exp=" + exp + " val=" + v);
                } while (false);
                do {
                    var ow = Context.newByteArrayObjectWriter(codec);
                    ow.write(v, v);
                    var ba = ow.toByteArray();
                    Context.require((exp + exp).equals(hex(ba)), "encoding fail in test exp=" + exp + " val=" + v);
                    var or = Context.newByteArrayObjectReader(codec, ba);
                    Context.require(Arrays.equals(v, or.read(byte[].class)), "decoding fail in test exp=" + exp + " val=" + v);
                    Context.require(Arrays.equals(v, or.read(byte[].class)), "decoding fail in test exp=" + exp + " val=" + v);
                } while (false);
            }

            @Override
            public void assertListCodingEquals(String exp, byte[] v) {
                do {
                    var ow = Context.newByteArrayObjectWriter(codec);
                    if (v == null) {
                        ow.beginList(0);
                        ow.end();
                    } else {
                        ow.beginList(1);
                        ow.write(v);
                        ow.end();
                    }
                    var ba = ow.toByteArray();
                    Context.require(exp.equals(hex(ba)), "encoding fail in test exp=" + exp + " val=" + v);
                    var or = Context.newByteArrayObjectReader(codec, ba);
                    or.beginList();
                    if (v != null) {
                        Context.require(Arrays.equals(v, or.readByteArray()), "decoding fail in test exp=" + exp + " val=" + v);
                    }
                    or.end();
                } while (false);
            }
        }

        static <TH extends Throwable> void assertThrows(Class<TH> exp, Runnable r) {
            boolean throwable = false;
            try {
                r.run();
            } catch (Throwable e) {
                Context.require(e.getClass().getName().equals(exp.getName()), "unexpected exception thrown exp=" + exp.getName() + " actual=" + e.getClass().getName());
                throwable = true;
            }
            Context.require(throwable, "nothing is thrown");
        }

        public void testRLPNullable() {
            assertThrows(UnsupportedOperationException.class, new Runnable() {
                public void run() {
                    var ow = Context.newByteArrayObjectWriter("RLP");
                    ow.writeNull();
                }
            });
            assertThrows(UnsupportedOperationException.class, new Runnable() {
                public void run() {
                    var ow = Context.newByteArrayObjectWriter("RLP");
                    ow.writeNullable(0);
                }
            });
            assertThrows(UnsupportedOperationException.class, new Runnable() {
                public void run() {
                    var ow = Context.newByteArrayObjectWriter("RLPn");
                    ow.writeNullable(1);
                    var ba = ow.toByteArray();
                    var or = Context.newByteArrayObjectReader("RLP", ba);
                    or.readNullable(Integer.class);
                }
            });
            assertThrows(UnsupportedOperationException.class, new Runnable() {
                public void run() {
                    var ow = Context.newByteArrayObjectWriter("RLPn");
                    ow.writeNullable(1);
                    var ba = ow.toByteArray();
                    var or = Context.newByteArrayObjectReader("RLP", ba);
                    or.readNullableOrDefault(Integer.class, 1);
                }
            });
            assertThrows(UnsupportedOperationException.class, new Runnable() {
                public void run() {
                    var ow = Context.newByteArrayObjectWriter("RLP");
                    ow.writeNullable(0, 0);
                }
            });
            assertThrows(UnsupportedOperationException.class, new Runnable() {
                public void run() {
                    var ow = Context.newByteArrayObjectWriter("RLP");
                    ow.beginNullableList(0);
                    ow.end();
                }
            });
            assertThrows(UnsupportedOperationException.class, new Runnable() {
                public void run() {
                    var ow = Context.newByteArrayObjectWriter("RLPn");
                    ow.beginList(0);
                    ow.end();
                    var ba = ow.toByteArray();
                    var or = Context.newByteArrayObjectReader("RLP", ba);
                    or.beginNullableList();
                }
            });
            assertThrows(UnsupportedOperationException.class, new Runnable() {
                public void run() {
                    var ow = Context.newByteArrayObjectWriter("RLP");
                    ow.beginNullableMap(0);
                    ow.end();
                }
            });
            assertThrows(UnsupportedOperationException.class, new Runnable() {
                public void run() {
                    var ow = Context.newByteArrayObjectWriter("RLPn");
                    ow.beginMap(0);
                    ow.end();
                    var ba = ow.toByteArray();
                    var or = Context.newByteArrayObjectReader("RLP", ba);
                    or.beginNullableMap();
                }
            });
            assertThrows(UnsupportedOperationException.class, new Runnable() {
                public void run() {
                    var ow = Context.newByteArrayObjectWriter("RLP");
                    ow.writeListOfNullable(0, 0);
                    ow.end();
                }
            });
        }

        public static String repeat(String s, int n) {
            var sb = new StringBuilder();
            for (int i = 0; i < n; i++) {
                sb.append(s);
            }
            return sb.toString();
        }

        public static class Person {
            private final String name;
            private final int age;

            public Person(String name, int age) {
                this.name = name;
                this.age = age;
            }

            public String name() {
                return name;
            }

            public int age() {
                return age;
            }

            public static void writeObject(ObjectWriter ow, Person p) {
                ow.writeListOf(p.name, p.age);
            }

            public static Person readObject(ObjectReader or) {
                or.beginList();
                var name = or.readString();
                var age = or.readInt();
                or.end();
                return new Person(name, age);
            }

            @Override
            public boolean equals(Object o) {
                if (this == o) return true;
                if (o == null || getClass() != o.getClass()) return false;

                Person person = (Person) o;

                if (age != person.age) return false;
                return (name == person.name) || (name != null && name.equals(person.name));
            }
        }

        void testRLPCommonExtra(String c) {
            ByteArrayObjectWriter ow;
            ObjectReader or;

            ow = Context.newByteArrayObjectWriter(c);
            ow.writeListOf(1, 2);
            Context.require("c20102".equals(hex(ow.toByteArray())));
            or = Context.newByteArrayObjectReader(c, ow.toByteArray());
            or.beginList();
            Context.require(or.read(Integer.class).equals(1));
            Context.require(or.read(Integer.class).equals(2));
            or.end();

            ow = Context.newByteArrayObjectWriter(c);
            ow.beginList(2);
            ow.write(1);
            ow.write(2);
            ow.end();
            Context.require("c20102".equals(hex(ow.toByteArray())));
            or = Context.newByteArrayObjectReader(c, ow.toByteArray());
            or.beginList();
            Context.require(or.read(Integer.class).equals(1));
            Context.require(or.read(Integer.class).equals(2));
            or.end();

            ow = Context.newByteArrayObjectWriter(c);
            ow.beginMap(2);
            ow.write(1);
            ow.write(2);
            ow.end();
            Context.require("c20102".equals(hex(ow.toByteArray())));
            or = Context.newByteArrayObjectReader(c, ow.toByteArray());
            or.beginMap();
            Context.require(or.read(Integer.class).equals(1));
            Context.require(or.read(Integer.class).equals(2));
            or.end();

            ow = Context.newByteArrayObjectWriter(c);
            var addr = new Address(new byte[21]);
            ow.write(addr);
            Context.require(("95" + hex(addr.toByteArray())).equals(hex(ow.toByteArray())));
            or = Context.newByteArrayObjectReader(c, ow.toByteArray());
            Context.require(addr.equals(or.readAddress()));

            ow = Context.newByteArrayObjectWriter(c);
            ow.write((Object) addr);
            Context.require(("95" + hex(addr.toByteArray())).equals(hex(ow.toByteArray())));
            or = Context.newByteArrayObjectReader(c, ow.toByteArray());
            Context.require(addr.equals(or.read(Address.class)));

            ow = Context.newByteArrayObjectWriter(c);
            var p = new Person("abc", 10);
            ow.write(p);
            Context.require("c5836162630a".equals(hex(ow.toByteArray())));
            or = Context.newByteArrayObjectReader(c, ow.toByteArray());
            Context.require(p.equals(or.read(Person.class)));

            ow = Context.newByteArrayObjectWriter(c);
            ow.beginList(1);
            ow.write(1);
            ow.end();
            Context.require("c101".equals(hex(ow.toByteArray())));
            or = Context.newByteArrayObjectReader(c, ow.toByteArray());
            or.beginList();
            Context.require(or.read(Integer.class).equals(1));
            Context.require(or.readOrDefault(Integer.class, 2).equals(2));
            or.end();

            ow = Context.newByteArrayObjectWriter(c);
            ow.beginList(2);
            ow.write(1);
            ow.write(2);
            ow.end();
            Context.require("c20102".equals(hex(ow.toByteArray())));
            or = Context.newByteArrayObjectReader(c, ow.toByteArray());
            or.beginList();
            or.skip();
            Context.require(or.read(Integer.class).equals(2));
            Context.require(!or.hasNext());
            or.end();

            ow = Context.newByteArrayObjectWriter(c);
            ow.beginList(2);
            ow.write(1);
            ow.write(2);
            ow.end();
            Context.require("c20102".equals(hex(ow.toByteArray())));
            or = Context.newByteArrayObjectReader(c, ow.toByteArray());
            or.beginList();
            or.skip(2);
            Context.require(!or.hasNext());
            or.end();

        }

        public void testBadSource(String c) {
            Exception ex = null;
            try {
                var ow = Context.newByteArrayObjectWriter(c);
                ow.write(new byte[10]);
                var ba = ow.toByteArray();
                ba = Arrays.copyOfRange(ba, 0, ba.length - 1);
                var or = Context.newByteArrayObjectReader(c, ba);
                or.readByteArray();
            } catch (Exception e) {
                ex = e;
            }
            Context.require(ex != null);
        }

        @External
        public void testRLP() {
            RLPCodecTest.testRLPSimpleSmall(new RLPAssertion("RLP"));
            testRLPNullable();
            testRLPCommonExtra("RLP");
            testBadSource("RLP");
        }

        public void testRLPNNullable() {
            final String c = "RLPn";
            ByteArrayObjectWriter ow;
            ObjectReader or;

            ow = Context.newByteArrayObjectWriter(c);
            ow.writeNull();
            Context.require("f800".equals(hex(ow.toByteArray())));
            or = Context.newByteArrayObjectReader(c, ow.toByteArray());
            var v = or.readNullable(int.class);
            Context.require(v == null);

            ow = Context.newByteArrayObjectWriter(c);
            ow.writeNullable(0);
            Context.require("00".equals(hex(ow.toByteArray())));
            or = Context.newByteArrayObjectReader(c, ow.toByteArray());
            Context.require(or.readNullable(Integer.class).equals(0));

            ow = Context.newByteArrayObjectWriter(c);
            ow.writeNullable((Integer) null);
            Context.require("f800".equals(hex(ow.toByteArray())));
            or = Context.newByteArrayObjectReader(c, ow.toByteArray());
            Context.require(or.readNullable(Integer.class) == null);

            ow = Context.newByteArrayObjectWriter(c);
            ow.writeNullable(null, null);
            Context.require("f800f800".equals(hex(ow.toByteArray())));
            or = Context.newByteArrayObjectReader(c, ow.toByteArray());
            Context.require(or.readNullable(Integer.class) == null);
            Context.require(or.readNullable(Integer.class) == null);

            ow = Context.newByteArrayObjectWriter(c);
            ow.writeNullable(0, 0);
            Context.require("0000".equals(hex(ow.toByteArray())));
            or = Context.newByteArrayObjectReader(c, ow.toByteArray());
            Context.require(or.readNullable(Integer.class).equals(0));
            Context.require(or.readNullable(Integer.class).equals(0));

            ow = Context.newByteArrayObjectWriter(c);
            ow.beginNullableList(0);
            ow.write(0);
            ow.end();
            Context.require("c100".equals(hex(ow.toByteArray())));
            or = Context.newByteArrayObjectReader(c, ow.toByteArray());
            or.beginNullableList();
            Context.require(or.readNullable(Integer.class).equals(0));
            or.end();

            ow = Context.newByteArrayObjectWriter(c);
            ow.beginNullableMap(0);
            ow.write(0);
            ow.write(0);
            ow.end();
            Context.require("c20000".equals(hex(ow.toByteArray())));
            or = Context.newByteArrayObjectReader(c, ow.toByteArray());
            or.beginNullableMap();
            Context.require(or.readNullable(Integer.class).equals(0));
            Context.require(or.readNullable(Integer.class).equals(0));
            or.end();

            ow = Context.newByteArrayObjectWriter(c);
            ow.writeListOfNullable(0, 0);
            Context.require("c20000".equals(hex(ow.toByteArray())));
            or = Context.newByteArrayObjectReader(c, ow.toByteArray());
            or.beginNullableList();
            Context.require(or.readNullable(Integer.class).equals(0));
            Context.require(or.readNullable(Integer.class).equals(0));
            or.end();

            ow = Context.newByteArrayObjectWriter(c);
            ow.beginMap(2);
            ow.writeNullable(1);
            ow.end();
            Context.require("c101".equals(hex(ow.toByteArray())));
            or = Context.newByteArrayObjectReader(c, ow.toByteArray());
            or.beginMap();
            Context.require(or.readNullable(Integer.class).equals(1));
            Context.require(or.readNullableOrDefault(Integer.class, 2).equals(2));
            or.end();
        }

        @External
        public void testRLPN() {
            RLPNCodecTest.testRLPNSimpleSmall(new RLPAssertion("RLPn"));
            testRLPNNullable();
            testRLPCommonExtra("RLPn");
            testBadSource("RLPn");
        }
    }

    @Test
    void testRLP() {
        var c = sm.mustDeploy(
                new Class[]{Score.class, RLPCodecTest.class, RLPNCodecTest.class}
        );
        c.invoke("testRLP");
    }

    @Test
    void testRLPN() {
        var c = sm.mustDeploy(
                new Class[]{Score.class, RLPCodecTest.class, RLPNCodecTest.class}
        );
        c.invoke("testRLPN");
    }
}
