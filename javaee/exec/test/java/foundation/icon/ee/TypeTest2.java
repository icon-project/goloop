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

package foundation.icon.ee;

import foundation.icon.ee.test.ContractAddress;
import foundation.icon.ee.test.NoDebugTest;
import foundation.icon.ee.util.Strings;
import org.junit.jupiter.api.Assertions;
import org.junit.jupiter.api.Test;
import score.Address;
import score.Context;
import score.annotation.External;
import score.annotation.Optional;

import java.math.BigInteger;
import java.util.Arrays;
import java.util.HashMap;
import java.util.List;
import java.util.Map;
import java.util.Objects;


public class TypeTest2 extends NoDebugTest {
    public static class Person {
        private String name;
        private int age;

        public Person() {
        }

        public Person(String name, int age) {
            this.name = name;
            this.age = age;
        }

        public String getName() {
            return name;
        }

        public int getAge() {
            return age;
        }

        public void setName(String name) {
            this.name = name;
        }

        public void setAge(int age) {
            this.age = age;
        }
    }

    public static class PersonF {
        public String name;
        public int age;
    }

    public static class Student extends Person {
        private String major;

        public Student() {
        }

        public Student(String name, int age, String major) {
            super(name, age);
            this.major = major;
        }

        public String getMajor() {
            return major;
        }

        public void setMajor(String major) {
            this.major = major;
        }
    }

    public static class StudentF extends PersonF {
        public String major;
    }

    public static class Course {
        private Person teacher;
        private Student[] students;

        public Course() {
        }

        public Course(Person teacher, Student[] students) {
            this.teacher = teacher;
            this.students = students;
        }

        public Person getTeacher() {
            return teacher;
        }

        public void setTeacher(Person teacher) {
            this.teacher = teacher;
        }

        public Student[] getStudents() {
            return students;
        }

        public void setStudents(Student[] students) {
            this.students = students;
        }
    }

    public static class CourseF {
        public PersonF teacher;
        public StudentF[] students;
    }

    public static class Score {
        @External
        public boolean mboolean(boolean v) {
            return v;
        }

        @External
        public byte mbyte(byte v) {
            return v;
        }

        @External
        public char mchar(char v) {
            return v;
        }

        @External
        public short mshort(short v) {
            return v;
        }

        @External
        public int mint(int v) {
            return v;
        }

        @External
        public long mlong(long v) {
            return v;
        }

        @External
        public BigInteger mBigInteger(@Optional BigInteger v) {
            return v;
        }

        @External
        public String mString(@Optional String v) {
            return v;
        }

        @External
        public Address mAddress(@Optional Address v) {
            return v;
        }

        @External
        public Person mPerson(Person v) {
            return v;
        }

        @External
        public Student mStudent(Student v) {
            return v;
        }

        @External
        public Course mCourse(Course v) {
            return v;
        }

        @External
        public PersonF mPersonF(PersonF v) {
            return v;
        }

        @External
        public StudentF mStudentF(StudentF v) {
            return v;
        }

        @External
        public CourseF mCourseF(CourseF v) {
            return v;
        }

        @External
        public void mvoid() {
        }


        @External
        public boolean[] mbooleanArray(@Optional boolean[] v) {
            return v;
        }

        @External
        public byte[] mbyteArray(@Optional byte[] v) {
            return v;
        }

        @External
        public char[] mcharArray(@Optional char[] v) {
            return v;
        }

        @External
        public short[] mshortArray(@Optional short[] v) {
            return v;
        }

        @External
        public int[] mintArray(@Optional int[] v) {
            return v;
        }

        @External
        public long[] mlongArray(@Optional long[] v) {
            return v;
        }

        @External
        public BigInteger[] mBigIntegerArray(@Optional BigInteger[] v) {
            return v;
        }

        @External
        public String[] mStringArray(@Optional String[] v) {
            return v;
        }

        @External
        public Address[] mAddressArray(@Optional Address[] v) {
            return v;
        }

        @External
        public Person[] mPersonArray(@Optional Person[] v) {
            return v;
        }

        @External
        public Student[] mStudentArray(@Optional Student[] v) {
            return v;
        }

        @External
        public Course[] mCourseArray(Course[] v) {
            return v;
        }

        @External
        public PersonF[] mPersonFArray(@Optional PersonF[] v) {
            return v;
        }

        @External
        public StudentF[] mStudentFArray(@Optional StudentF[] v) {
            return v;
        }

        @External
        public CourseF[] mCourseFArray(CourseF[] v) {
            return v;
        }


        @External
        public boolean[][] mbooleanArray2D(boolean[][] v) {
            return v;
        }

        @External
        public byte[][] mbyteArray2D(@Optional byte[][] v) {
            return v;
        }

        @External
        public char[][] mcharArray2D(char[][] v) {
            return v;
        }

        @External
        public short[][] mshortArray2D(short[][] v) {
            return v;
        }

        @External
        public int[][] mintArray2D(int[][] v) {
            return v;
        }

        @External
        public long[][] mlongArray2D(long[][] v) {
            return v;
        }

        @External
        public BigInteger[][] mBigIntegerArray2D(@Optional BigInteger[][] v) {
            return v;
        }

        @External
        public String[][] mStringArray2D(@Optional String[][] v) {
            return v;
        }

        @External
        public Address[][] mAddressArray2D(@Optional Address[][] v) {
            return v;
        }

        @External
        public Person[][] mPersonArray2D(Person[][] v) {
            return v;
        }

        @External
        public Student[][] mStudentArray2D(Student[][] v) {
            return v;
        }

        @External
        public Course[][] mCourseArray2D(Course[][] v) {
            return v;
        }

        @External
        public PersonF[][] mPersonFArray2D(PersonF[][] v) {
            return v;
        }

        @External
        public StudentF[][] mStudentFArray2D(StudentF[][] v) {
            return v;
        }

        @External
        public CourseF[][] mCourseFArray2D(CourseF[][] v) {
            return v;
        }


        @External
        public boolean[][][] mbooleanArray3D(boolean[][][] v) {
            return v;
        }

        @External
        public byte[][][] mbyteArray3D(@Optional byte[][][] v) {
            return v;
        }

        @External
        public char[][][] mcharArray3D(char[][][] v) {
            return v;
        }

        @External
        public short[][][] mshortArray3D(short[][][] v) {
            return v;
        }

        @External
        public int[][][] mintArray3D(int[][][] v) {
            return v;
        }

        @External
        public long[][][] mlongArray3D(long[][][] v) {
            return v;
        }

        @External
        public BigInteger[][][] mBigIntegerArray3D(@Optional BigInteger[][][] v) {
            return v;
        }

        @External
        public String[][][] mStringArray3D(@Optional String[][][] v) {
            return v;
        }

        @External
        public Address[][][] mAddressArray3D(@Optional Address[][][] v) {
            return v;
        }

        @External
        public Person[][][] mPersonArray3D(Person[][][] v) {
            return v;
        }

        @External
        public Student[][][] mStudentArray3D(Student[][][] v) {
            return v;
        }

        @External
        public Course[][][] mCourseArray3D(Course[][][] v) {
            return v;
        }

        @External
        public PersonF[][][] mPersonFArray3D(PersonF[][][] v) {
            return v;
        }

        @External
        public StudentF[][][] mStudentFArray3D(StudentF[][][] v) {
            return v;
        }

        @External
        public CourseF[][][] mCourseFArray3D(CourseF[][][] v) {
            return v;
        }

        @External
        public List<?> mFreeList() {
            return List.of(
                    "string",
                    1,
                    new Person("name", 1)
            );
        }

        @External
        public Map<?, ?> mFreeMap() {
            return Map.of(
                    "list", List.of(1, "string"),
                    "array", new Person[] { new Person("name1", 1)},
                    "struct", new Person("name1", 1)
            );
        }
    }

    public static class CallerScore {
        private final Address callee;

        public CallerScore(Address callee) {
            this.callee = callee;
        }

        @External
        public boolean mboolean(boolean v) {
            var vv = Context.call(boolean.class, callee, "mboolean", v);
            // assert vv != null : this makes ref to TypeTest2.class
            return vv;
        }

        @External
        public byte mbyte(byte v) {
            var vv = Context.call(byte.class, callee, "mbyte", v);
            return vv;
        }

        @External
        public char mchar(char v) {
            var vv = Context.call(char.class, callee, "mchar", v);
            return vv;
        }

        @External
        public short mshort(short v) {
            var vv = Context.call(short.class, callee, "mshort", v);
            return vv;
        }

        @External
        public int mint(int v) {
            var vv = Context.call(int.class, callee, "mint", v);
            return vv;
        }

        @External
        public long mlong(long v) {
            var vv = Context.call(long.class, callee, "mlong", v);
            return vv;
        }

        @External
        public BigInteger mBigInteger(BigInteger v) {
            return Context.call(BigInteger.class, callee, "mBigInteger", v);
        }

        @External
        public String mString(String v) {
            return Context.call(String.class, callee, "mString", v);
        }

        @External
        public Address mAddress(@Optional Address v) {
            return Context.call(Address.class, callee, "mAddress", v);
        }

        @External
        public Person mPerson(Person v) {
            return Context.call(Person.class, callee, "mPerson", v);
        }

        @External
        public Student mStudent(Student v) {
            return Context.call(Student.class, callee, "mStudent", v);
        }

        @External
        public Course mCourse(Course v) {
            return Context.call(Course.class, callee, "mCourse", v);
        }

        @External
        public PersonF mPersonF(PersonF v) {
            return Context.call(PersonF.class, callee, "mPersonF", v);
        }

        @External
        public StudentF mStudentF(StudentF v) {
            return Context.call(StudentF.class, callee, "mStudentF", v);
        }

        @External
        public CourseF mCourseF(CourseF v) {
            return Context.call(CourseF.class, callee, "mCourseF", v);
        }

        @External
        public boolean[] mbooleanArray(boolean[] v) {
            return Context.call(boolean[].class, callee, "mbooleanArray", (Object) v);
        }

        @External
        public byte[] mbyteArray(byte[] v) {
            return Context.call(byte[].class, callee, "mbyteArray", (Object) v);
        }

        @External
        public char[] mcharArray(char[] v) {
            return Context.call(char[].class, callee, "mcharArray", (Object) v);
        }

        @External
        public short[] mshortArray(short[] v) {
            return Context.call(short[].class, callee, "mshortArray", (Object) v);
        }

        @External
        public int[] mintArray(int[] v) {
            return Context.call(int[].class, callee, "mintArray", (Object) v);
        }

        @External
        public long[] mlongArray(long[] v) {
            return Context.call(long[].class, callee, "mlongArray", (Object) v);
        }

        @External
        public BigInteger[] mBigIntegerArray(BigInteger[] v) {
            return Context.call(BigInteger[].class, callee, "mBigIntegerArray", (Object) v);
        }

        @External
        public String[] mStringArray(String[] v) {
            return Context.call(String[].class, callee, "mStringArray", (Object) v);
        }

        @External
        public Address[] mAddressArray(Address[] v) {
            return Context.call(Address[].class, callee, "mAddressArray", (Object) v);
        }

        @External
        public Person[] mPersonArray(Person[] v) {
            return Context.call(Person[].class, callee, "mPersonArray", (Object) v);
        }

        @External
        public Student[] mStudentArray(Student[] v) {
            return Context.call(Student[].class, callee, "mStudentArray", (Object) v);
        }

        @External
        public Course[] mCourseArray(Course[] v) {
            return Context.call(Course[].class, callee, "mCourseArray", (Object) v);
        }

        @External
        public PersonF[] mPersonFArray(PersonF[] v) {
            return Context.call(PersonF[].class, callee, "mPersonFArray", (Object) v);
        }

        @External
        public StudentF[] mStudentFArray(StudentF[] v) {
            return Context.call(StudentF[].class, callee, "mStudentFArray", (Object) v);
        }

        @External
        public CourseF[] mCourseFArray(CourseF[] v) {
            return Context.call(CourseF[].class, callee, "mCourseFArray", (Object) v);
        }

        @External
        public void mvoid() {
            Context.call(callee, "mvoid");
        }

        @External
        public boolean[][] mbooleanArray2D(boolean[][] v) {
            return Context.call(boolean[][].class, callee, "mbooleanArray2D", (Object) v);
        }

        @External
        public byte[][] mbyteArray2D(byte[][] v) {
            return Context.call(byte[][].class, callee, "mbyteArray2D", (Object) v);
        }

        @External
        public char[][] mcharArray2D(char[][] v) {
            return Context.call(char[][].class, callee, "mcharArray2D", (Object) v);
        }

        @External
        public short[][] mshortArray2D(short[][] v) {
            return Context.call(short[][].class, callee, "mshortArray2D", (Object) v);
        }

        @External
        public int[][] mintArray2D(int[][] v) {
            return Context.call(int[][].class, callee, "mintArray2D", (Object) v);
        }

        @External
        public long[][] mlongArray2D(long[][] v) {
            return Context.call(long[][].class, callee, "mlongArray2D", (Object) v);
        }

        @External
        public BigInteger[][] mBigIntegerArray2D(BigInteger[][] v) {
            return Context.call(BigInteger[][].class, callee, "mBigIntegerArray2D", (Object) v);
        }

        @External
        public String[][] mStringArray2D(String[][] v) {
            return Context.call(String[][].class, callee, "mStringArray2D", (Object) v);
        }

        @External
        public Address[][] mAddressArray2D(Address[][] v) {
            return Context.call(Address[][].class, callee, "mAddressArray2D", (Object) v);
        }

        @External
        public Person[][] mPersonArray2D(Person[][] v) {
            return Context.call(Person[][].class, callee, "mPersonArray2D", (Object) v);
        }

        @External
        public Student[][] mStudentArray2D(Student[][] v) {
            return Context.call(Student[][].class, callee, "mStudentArray2D", (Object) v);
        }

        @External
        public Course[][] mCourseArray2D(Course[][] v) {
            return Context.call(Course[][].class, callee, "mCourseArray2D", (Object) v);
        }

        @External
        public PersonF[][] mPersonFArray2D(PersonF[][] v) {
            return Context.call(PersonF[][].class, callee, "mPersonFArray2D", (Object) v);
        }

        @External
        public StudentF[][] mStudentFArray2D(StudentF[][] v) {
            return Context.call(StudentF[][].class, callee, "mStudentFArray2D", (Object) v);
        }

        @External
        public CourseF[][] mCourseFArray2D(CourseF[][] v) {
            return Context.call(CourseF[][].class, callee, "mCourseFArray2D", (Object) v);
        }

        @External
        public boolean[][][] mbooleanArray3D(boolean[][][] v) {
            return Context.call(boolean[][][].class, callee, "mbooleanArray3D", (Object) v);
        }

        @External
        public byte[][][] mbyteArray3D(byte[][][] v) {
            return Context.call(byte[][][].class, callee, "mbyteArray3D", (Object) v);
        }

        @External
        public char[][][] mcharArray3D(char[][][] v) {
            return Context.call(char[][][].class, callee, "mcharArray3D", (Object) v);
        }

        @External
        public short[][][] mshortArray3D(short[][][] v) {
            return Context.call(short[][][].class, callee, "mshortArray3D", (Object) v);
        }

        @External
        public int[][][] mintArray3D(int[][][] v) {
            return Context.call(int[][][].class, callee, "mintArray3D", (Object) v);
        }

        @External
        public long[][][] mlongArray3D(long[][][] v) {
            return Context.call(long[][][].class, callee, "mlongArray3D", (Object) v);
        }

        @External
        public BigInteger[][][] mBigIntegerArray3D(BigInteger[][][] v) {
            return Context.call(BigInteger[][][].class, callee, "mBigIntegerArray3D", (Object) v);
        }

        @External
        public String[][][] mStringArray3D(String[][][] v) {
            return Context.call(String[][][].class, callee, "mStringArray3D", (Object) v);
        }

        @External
        public Address[][][] mAddressArray3D(Address[][][] v) {
            return Context.call(Address[][][].class, callee, "mAddressArray3D", (Object) v);
        }

        @External
        public Person[][][] mPersonArray3D(Person[][][] v) {
            return Context.call(Person[][][].class, callee, "mPersonArray3D", (Object) v);
        }

        @External
        public Student[][][] mStudentArray3D(Student[][][] v) {
            return Context.call(Student[][][].class, callee, "mStudentArray3D", (Object) v);
        }

        @External
        public Course[][][] mCourseArray3D(Course[][][] v) {
            return Context.call(Course[][][].class, callee, "mCourseArray3D", (Object) v);
        }

        @External
        public PersonF[][][] mPersonFArray3D(PersonF[][][] v) {
            return Context.call(PersonF[][][].class, callee, "mPersonFArray3D", (Object) v);
        }

        @External
        public StudentF[][][] mStudentFArray3D(StudentF[][][] v) {
            return Context.call(StudentF[][][].class, callee, "mStudentFArray3D", (Object) v);
        }

        @External
        public CourseF[][][] mCourseFArray3D(CourseF[][][] v) {
            return Context.call(CourseF[][][].class, callee, "mCourseFArray3D", (Object) v);
        }

        @External
        public List<?> mFreeList() {
            return Context.call(List.class, callee, "mFreeList");
        }

        @External
        public Map<?, ?> mFreeMap() {
            return Context.call(Map.class, callee, "mFreeMap");
        }
    }

    private void assertEquals(Object o1, Object o2) {
        Assertions.assertEquals(wrap(o1), wrap(o2));
    }

    private void test(ContractAddress app, Object object, String method) {
        Assertions.assertEquals(wrap(object), wrap(app.invoke(method, object).getRet()));
    }

    private Object[][][] nest3D(Object obj) {
        return new Object[][][] {
                new Object[][] {
                        new Object[] {
                                obj
                        }
                }
        };
    }

    private BigInteger bigInt(long v) {
        return BigInteger.valueOf(v);
    }

    private static class ArrayWrapper {
        private final Object value;

        public ArrayWrapper(Object value) {
            this.value = value;
        }

        @Override
        public boolean equals(Object o) {
            if (this == o) return true;
            if (o == null || getClass() != o.getClass()) return false;
            ArrayWrapper that = (ArrayWrapper) o;

            if (value instanceof Object[]) {
                if (!(that.value instanceof Object[])) {
                    return false;
                }
                var v = (Object[]) value;
                var thatV = (Object[]) that.value;
                return Arrays.equals(v, thatV);
            } else if (value instanceof byte[]) {
                if (!(that.value instanceof byte[])) {
                    return false;
                }
                var v = (byte[]) value;
                var thatV = (byte[]) that.value;
                return Arrays.equals(v, thatV);
            }
            return Objects.equals(value, that.value);
        }

        @Override
        public int hashCode() {
            return Objects.hash(value);
        }

        public String toString() {
            if (value instanceof Object[]) {
                return Arrays.asList((Object[])value).toString();
            } else if (value instanceof byte[]) {
                return "[" + Strings.hexFromBytes((byte[])value, " ") + "]";
            }
            return value.toString();
        }
    }

    private Object wrap(Object obj) {
        if (obj instanceof Object[]) {
            var o = (Object[]) obj;
            var res = new Object[o.length];
            for (int i = 0; i < o.length; ++i) {
                res[i] = wrap(o[i]);
            }
            return new ArrayWrapper(res);
        } else if (obj instanceof byte[]) {
            return new ArrayWrapper(obj);
        } else if (obj instanceof Map<?, ?>) {
            var res = new HashMap<>();
            for (var e : ((Map<?, ?>)obj).entrySet()) {
                res.put(e.getKey(), wrap(e.getValue()));
            }
            return res;
        }
        return obj;
    }

    @Test
    public void testParamAndReturn() {
        var app = sm.mustDeploy(new Class<?>[]{
                Score.class, Person.class, Student.class, Course.class,
                PersonF.class, StudentF.class, CourseF.class
        });
        testForApp(app);
    }

    @Test
    public void testInterncall() {
        var app = sm.mustDeploy(new Class<?>[]{
                Score.class, Person.class, Student.class, Course.class,
                PersonF.class, StudentF.class, CourseF.class
        });
        var caller = sm.mustDeploy(new Class<?>[]{
                CallerScore.class, Person.class, Student.class, Course.class,
                PersonF.class, StudentF.class, CourseF.class
        }, app.getAddress());
        testForApp(caller);
    }

    private void testForApp(ContractAddress app) {
        final var booleanArray3D = nest3D(true);
        final var byteArray3D = new Object[][] {
                new Object[] {
                        new byte[] {1}
                }
        };
        final var bigIntArray3D = nest3D(BigInteger.valueOf(1));
        final var stringArray3D = nest3D(" a string");
        final var addressArray3D = nest3D(
                new foundation.icon.ee.types.Address(new byte[Address.LENGTH])
        );
        final var personMapArray3D = nest3D(Map.of(
                "name", "name",
                "age", bigInt(10)
        ));
        final var studentMapArray3D = nest3D(Map.of(
                "name", "name",
                "age", bigInt(10),
                "major", "major"
        ));
        final var courseMapArray3D = nest3D(Map.of(
                "teacher", Map.of(
                        "name", "name1",
                        "age", bigInt(30)
                ),
                "students", new Object[] {
                        Map.of(
                                "name", "name2",
                                "age", bigInt(10),
                                "major", "m1"
                        ),
                        Map.of(
                                "name", "name3",
                                "age", bigInt(11),
                                "major", "m2"
                        )
                }
        ));
        test(app, booleanArray3D[0][0][0], "mboolean");
        test(app, bigIntArray3D[0][0][0], "mbyte");
        test(app, bigIntArray3D[0][0][0], "mchar");
        test(app, bigIntArray3D[0][0][0], "mshort");
        test(app, bigIntArray3D[0][0][0], "mint");
        test(app, bigIntArray3D[0][0][0], "mlong");
        test(app, bigIntArray3D[0][0][0], "mBigInteger");
        test(app, stringArray3D[0][0][0], "mString");
        test(app, addressArray3D[0][0][0], "mAddress");
        test(app, personMapArray3D[0][0][0], "mPerson");
        test(app, studentMapArray3D[0][0][0], "mStudent");
        test(app, courseMapArray3D[0][0][0], "mCourse");
        test(app, personMapArray3D[0][0][0], "mPersonF");
        test(app, studentMapArray3D[0][0][0], "mStudentF");
        test(app, courseMapArray3D[0][0][0], "mCourseF");

        test(app, booleanArray3D[0][0], "mbooleanArray");
        test(app, byteArray3D[0][0], "mbyteArray");
        test(app, bigIntArray3D[0][0], "mcharArray");
        test(app, bigIntArray3D[0][0], "mshortArray");
        test(app, bigIntArray3D[0][0], "mintArray");
        test(app, bigIntArray3D[0][0], "mlongArray");
        test(app, bigIntArray3D[0][0], "mBigIntegerArray");
        test(app, stringArray3D[0][0], "mStringArray");
        test(app, addressArray3D[0][0], "mAddressArray");
        test(app, personMapArray3D[0][0], "mPersonArray");
        test(app, studentMapArray3D[0][0], "mStudentArray");
        test(app, courseMapArray3D[0][0], "mCourseArray");
        test(app, personMapArray3D[0][0], "mPersonFArray");
        test(app, studentMapArray3D[0][0], "mStudentFArray");
        test(app, courseMapArray3D[0][0], "mCourseFArray");
        Assertions.assertNull(app.invoke("mvoid").getRet());

        test(app, booleanArray3D[0], "mbooleanArray2D");
        test(app, byteArray3D[0], "mbyteArray2D");
        test(app, bigIntArray3D[0], "mcharArray2D");
        test(app, bigIntArray3D[0], "mshortArray2D");
        test(app, bigIntArray3D[0], "mintArray2D");
        test(app, bigIntArray3D[0], "mlongArray2D");
        test(app, bigIntArray3D[0], "mBigIntegerArray2D");
        test(app, stringArray3D[0], "mStringArray2D");
        test(app, addressArray3D[0], "mAddressArray2D");
        test(app, personMapArray3D[0], "mPersonArray2D");
        test(app, studentMapArray3D[0], "mStudentArray2D");
        test(app, courseMapArray3D[0], "mCourseArray2D");
        test(app, personMapArray3D[0], "mPersonFArray2D");
        test(app, studentMapArray3D[0], "mStudentFArray2D");
        test(app, courseMapArray3D[0], "mCourseFArray2D");

        test(app, booleanArray3D, "mbooleanArray3D");
        test(app, byteArray3D, "mbyteArray3D");
        test(app, bigIntArray3D, "mcharArray3D");
        test(app, bigIntArray3D, "mshortArray3D");
        test(app, bigIntArray3D, "mintArray3D");
        test(app, bigIntArray3D, "mlongArray3D");
        test(app, bigIntArray3D, "mBigIntegerArray3D");
        test(app, stringArray3D, "mStringArray3D");
        test(app, addressArray3D, "mAddressArray3D");
        test(app, personMapArray3D, "mPersonArray3D");
        test(app, studentMapArray3D, "mStudentArray3D");
        test(app, courseMapArray3D, "mCourseArray3D");
        test(app, personMapArray3D, "mPersonFArray3D");
        test(app, studentMapArray3D, "mStudentFArray3D");
        test(app, courseMapArray3D, "mCourseFArray3D");

        var freeList = new Object[]{
                "string",
                BigInteger.valueOf(1),
                Map.of(
                        "name", "name",
                        "age", BigInteger.valueOf(1)
                )
        };
        assertEquals(freeList, app.invoke("mFreeList").getRet());
        var freeMap = Map.of(
                "list", new Object[] { bigInt(1), "string"},
                "array", new Object[] {
                        Map.of(
                                "name", "name1",
                                "age", bigInt(1)
                        )
                },
                "struct", Map.of(
                        "name", "name1",
                        "age", bigInt(1)
                )
        );
        assertEquals(freeMap, app.invoke("mFreeMap").getRet());
    }
}
