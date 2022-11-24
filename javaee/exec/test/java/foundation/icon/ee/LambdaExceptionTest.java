/*
 * Copyright 2022 ICON Foundation
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

import foundation.icon.ee.test.SimpleTest;
import foundation.icon.ee.test.TransactionException;
import foundation.icon.ee.types.Status;
import org.junit.jupiter.api.Assertions;
import org.junit.jupiter.api.Test;

import java.io.IOException;
import java.math.BigInteger;
import java.util.function.Function;

import org.objectweb.asm.AnnotationVisitor;
import org.objectweb.asm.ClassWriter;
import org.objectweb.asm.FieldVisitor;
import org.objectweb.asm.Handle;
import org.objectweb.asm.MethodVisitor;
import org.objectweb.asm.Opcodes;
import org.objectweb.asm.Type;
import score.Context;
import score.RevertedException;
import score.annotation.External;

public class LambdaExceptionTest extends SimpleTest {
    public static class RunnableScore {
        @External
        public void run() {
            Runnable f = () -> {
                throw new IllegalArgumentException();
            };
            var expected = false;
            try {
                f.run();
                Context.require(false, "no exception");
            } catch (IllegalArgumentException e) {
                expected = true;
            }
            Context.require(expected, "not an IllegalArgumentException");
        }
    }

    @Test
    void runnable() {
        var c = sm.mustDeploy(RunnableScore.class);
        c.invoke("run");
    }

    public static class NestedRunnableScore {
        @External
        public void run() {
            Runnable f = () -> {
                Runnable g = () -> {
                    throw new IllegalArgumentException();
                };
                g.run();
            };
            var expected = false;
            try {
                f.run();
                Context.require(false, "no exception");
            } catch (IllegalArgumentException e) {
                expected = true;
            }
            Context.require(expected, "not an IllegalArgumentException");
        }
    }

    @Test
    void nestedRunnable() {
        var c = sm.mustDeploy(NestedRunnableScore.class);
        c.invoke("run");
    }

    public static class FunctionScore {
        @External
        public void run() {
            Function<BigInteger, BigInteger> f = (a) -> {
                throw new IllegalArgumentException();
            };
            var expected = false;
            try {
                f.apply(BigInteger.ONE);
                Context.require(false, "no exception");
            } catch (IllegalArgumentException e) {
                expected = true;
            }
            Context.require(expected, "not an IllegalArgumentException");
        }
    }

    @Test
    void function() {
        var c = sm.mustDeploy(FunctionScore.class);
        c.invoke("run");
    }

    public static class NestedFunctionScore {
        @External
        public void run() {
            Function<BigInteger, BigInteger> f = (a) -> {
                Function<BigInteger, BigInteger> g = (b) -> {
                    throw new IllegalArgumentException();
                };
                return g.apply(a);
            };
            var expected = false;
            try {
                f.apply(BigInteger.ONE);
                Context.require(false, "no exception");
            } catch (IllegalArgumentException e) {
                expected = true;
            }
            Context.require(expected, "not an IllegalArgumentException");
        }
    }

    @Test
    void nestedFunction() {
        var c = sm.mustDeploy(FunctionScore.class);
        c.invoke("run");
    }

    public static class AvmExceptionScore {
        @External
        public void revert() {
            Runnable f = () -> {
                try {
                    // throws AvmException
                    Context.require(false, "AvmException by force");
                } catch (Throwable t) {
                    // cannot catch
                    Context.require(false, "shall not caught AvmException");
                }
            };
            f.run();
            Context.require(false, "no exception");
        }

        @External
        public void run() {
            var expected = false;
            try {
                Context.call(Context.getAddress(), "revert");
                Context.require(false, "no exception");
            } catch (RevertedException e) {
                expected = true;
            }
            Context.require(expected, "not a RevertedException");
        }
    }

    @Test
    void avmException() {
        var c = sm.mustDeploy(AvmExceptionScore.class);
        c.invoke("run");
    }

    /*
package jtest;

import score.annotation.External;

public class App {
    @External
    public void f(){
        Runnable r = () -> {
            throw new ReflectiveOperationException();
        };
        r.run();
    }
}
     */
    public static class AppDump implements Opcodes {
        public static byte[] dump() {
            ClassWriter classWriter = new ClassWriter(0);
            FieldVisitor fieldVisitor;
            MethodVisitor methodVisitor;
            AnnotationVisitor annotationVisitor0;

            classWriter.visit(V11, ACC_PUBLIC | ACC_SUPER, "jtest/App", null, "java/lang/Object", null);

            classWriter.visitInnerClass("java/lang/invoke/MethodHandles$Lookup", "java/lang/invoke/MethodHandles", "Lookup", ACC_PUBLIC | ACC_FINAL | ACC_STATIC);

            {
                methodVisitor = classWriter.visitMethod(ACC_PUBLIC, "<init>", "()V", null, null);
                methodVisitor.visitCode();
                methodVisitor.visitVarInsn(ALOAD, 0);
                methodVisitor.visitMethodInsn(INVOKESPECIAL, "java/lang/Object", "<init>", "()V", false);
                methodVisitor.visitInsn(RETURN);
                methodVisitor.visitMaxs(1, 1);
                methodVisitor.visitEnd();
            }
            {
                methodVisitor = classWriter.visitMethod(ACC_PUBLIC, "f", "()V", null, null);
                {
                    annotationVisitor0 = methodVisitor.visitAnnotation("Lscore/annotation/External;", false);
                    annotationVisitor0.visitEnd();
                }
                methodVisitor.visitCode();
                methodVisitor.visitInvokeDynamicInsn("run", "()Ljava/lang/Runnable;", new Handle(Opcodes.H_INVOKESTATIC, "java/lang/invoke/LambdaMetafactory", "metafactory", "(Ljava/lang/invoke/MethodHandles$Lookup;Ljava/lang/String;Ljava/lang/invoke/MethodType;Ljava/lang/invoke/MethodType;Ljava/lang/invoke/MethodHandle;Ljava/lang/invoke/MethodType;)Ljava/lang/invoke/CallSite;", false), new Object[]{Type.getType("()V"), new Handle(Opcodes.H_INVOKESTATIC, "jtest/App", "lambda$f$0", "()V", false), Type.getType("()V")});
                methodVisitor.visitVarInsn(ASTORE, 1);
                methodVisitor.visitVarInsn(ALOAD, 1);
                methodVisitor.visitMethodInsn(INVOKEINTERFACE, "java/lang/Runnable", "run", "()V", true);
                methodVisitor.visitInsn(RETURN);
                methodVisitor.visitMaxs(1, 2);
                methodVisitor.visitEnd();
            }
            {
                methodVisitor = classWriter.visitMethod(ACC_PRIVATE | ACC_STATIC | ACC_SYNTHETIC, "lambda$f$0", "()V", null, null);
                methodVisitor.visitCode();
                methodVisitor.visitTypeInsn(NEW, "java/lang/ReflectiveOperationException");
                methodVisitor.visitInsn(DUP);
                methodVisitor.visitMethodInsn(INVOKESPECIAL, "java/lang/ReflectiveOperationException", "<init>", "()V", false);
                methodVisitor.visitInsn(ATHROW);
                methodVisitor.visitMaxs(2, 0);
                methodVisitor.visitEnd();
            }
            classWriter.visitEnd();

            return classWriter.toByteArray();
        }
    }

    @Test
    void nonRuntimeException() {
        var bc = AppDump.dump();
        var c = sm.mustDeploy("jtest.App", bc);
        var res = c.tryInvoke("f");
        Assertions.assertEquals(Status.UnknownFailure, res.getStatus());
        // rerun to check EE is still alive
        res = c.tryInvoke("f");
        Assertions.assertEquals(Status.UnknownFailure, res.getStatus());
    }
}
