package org.aion.avm.core;

import org.objectweb.asm.ClassReader;
import org.objectweb.asm.ClassVisitor;
import org.objectweb.asm.ClassWriter;

import java.util.ArrayList;
import java.util.List;
import java.util.Objects;

/**
 * @author Roman Katerinenko
 */
public final class ClassToolchain {
    private final ClassReader reader;
    private final ClassWriter writer;
    private final ClassVisitor classVisitor;
    private final int parsingOptions;

    private ClassToolchain(byte[] bytecode, ClassVisitor visitor, ClassWriter writer, int parsingOptions) {
        this.reader = new ClassReader(bytecode);
        this.classVisitor = visitor;
        this.writer = writer;
        this.parsingOptions = parsingOptions;
    }

    public byte[] runAndGetBytecode() {
        reader.accept(classVisitor, parsingOptions);
        return writer.toByteArray();
    }

    public static final class Builder {
        private final byte[] bytecode;
        private final List<ToolChainClassVisitor> visitorSequence = new ArrayList<>();
        private final int parsingOptions;

        private ClassWriter writer;

        public Builder(byte[] bytecode, int parsingOptions) {
            Objects.requireNonNull(bytecode);
            this.bytecode = bytecode;
            this.parsingOptions = parsingOptions;
        }

        public Builder addNextVisitor(ToolChainClassVisitor visitor) {
            Objects.requireNonNull(visitor);
            visitorSequence.add(visitor);
            return this;
        }

        public Creator addWriter(ClassWriter writer) {
            Objects.requireNonNull(writer);
            this.writer = writer;
            return new Creator();
        }

        public final class Creator {
            public ClassToolchain build() {
                ClassVisitor prevVisitor = writer;
                for (int i = visitorSequence.size() - 1; i >= 0; i--) {
                    ToolChainClassVisitor curVisitor = visitorSequence.get(i);
                    curVisitor.setDelegate(prevVisitor);
                    prevVisitor = curVisitor;
                }
                return new ClassToolchain(bytecode, visitorSequence.get(0), writer, parsingOptions);
            }
        }
    }

    public static class ToolChainClassVisitor extends ClassVisitor {

        protected ToolChainClassVisitor(int api) {
            super(api, null);
        }

        /**
         * We need to open up access to the setDelegate() so unit tests can still operate on components as though they were part of a
         * standard ASM pipeline for some testing scenarios which are really only possible if we can issue the callbacks, directly,
         * instead of starting with class bytes.
         * NOTE:  Should only be used for testing.
         */
        public void setDelegate(ClassVisitor delegate) {
            this.cv = delegate;
        }

    }

}