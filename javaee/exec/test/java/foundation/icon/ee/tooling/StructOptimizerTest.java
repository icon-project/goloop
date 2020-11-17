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

package foundation.icon.ee.tooling;

import org.junit.jupiter.api.Test;
import score.annotation.External;

public class StructOptimizerTest extends OptimizerGoldenTest {
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

    public static class ScoreParamPerson {
        @External
        public void method(Person p) {
        }
    }

    @Test
    void paramPerson() {
        test(ScoreParamPerson.class, Person.class, Student.class, Course.class);
    }

    public static class ScoreParamStudent {
        @External
        public void method(Student p) {
        }
    }

    @Test
    void paramStudent() {
        test(ScoreParamStudent.class, Person.class, Student.class, Course.class);
    }

    public static class ScoreParamCourse {
        @External
        public void method(Course p) {
        }
    }

    @Test
    void paramCourse() {
        test(ScoreParamCourse.class, Person.class, Student.class, Course.class);
    }

    public static class ScoreReturnPerson {
        @External
        public Person method() {
            return new Person("name", 10);
        }
    }

    @Test
    void returnPerson() {
        test(ScoreReturnPerson.class, Person.class, Student.class, Course.class);
    }

    public static class ScoreReturnStudent {
        @External
        public Student method() {
            return new Student("name", 10, "major");
        }
    }

    @Test
    void returnStudent() {
        test(ScoreReturnStudent.class, Person.class, Student.class, Course.class);
    }

    public static class ScoreReturnCourse {
        @External
        public Course method() {
            return new Course();
        }
    }

    @Test
    void returnCourse() {
        test(ScoreReturnCourse.class, Person.class, Student.class, Course.class);
    }

    public static class ScoreParamPersonF {
        @External
        public void method(PersonF p) {
        }
    }

    @Test
    void paramPersonF() {
        test(ScoreParamPersonF.class, PersonF.class, StudentF.class, CourseF.class);
    }

    public static class ScoreParamStudentF {
        @External
        public void method(StudentF p) {
        }
    }

    @Test
    void paramStudentF() {
        test(ScoreParamStudentF.class, PersonF.class, StudentF.class, CourseF.class);
    }

    public static class ScoreParamCourseF {
        @External
        public void method(CourseF p) {
        }
    }

    @Test
    void paramCourseF() {
        test(ScoreParamCourseF.class, PersonF.class, StudentF.class, CourseF.class);
    }

    public static class ScoreReturnPersonF {
        @External
        public PersonF method() {
            var ret = new PersonF();
            ret.name = "name";
            ret.age = 10;
            return ret;
        }
    }

    @Test
    void returnPersonF() {
        test(ScoreReturnPersonF.class, PersonF.class, StudentF.class, CourseF.class);
    }

    public static class ScoreReturnStudentF {
        @External
        public StudentF method() {
            var ret = new StudentF();
            ret.name = "name";
            ret.age = 10;
            ret.major = "major";
            return ret;
        }
    }

    @Test
    void returnStudentF() {
        test(ScoreReturnStudentF.class, PersonF.class, StudentF.class, CourseF.class);
    }

    public static class ScoreReturnCourseF {
        @External
        public CourseF method() {
            return new CourseF();
        }
    }

    @Test
    void returnCourseF() {
        test(ScoreReturnCourseF.class, PersonF.class, StudentF.class, CourseF.class);
    }
}
