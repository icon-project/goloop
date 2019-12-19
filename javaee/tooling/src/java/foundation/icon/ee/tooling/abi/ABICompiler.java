/*
 * Copyright 2019 ICON Foundation
 * Copyright (c) 2018 Aion Foundation https://aion.network/
 */

package foundation.icon.ee.tooling.abi;

import foundation.icon.ee.types.Method;
import org.aion.avm.utilities.JarBuilder;
import org.aion.avm.utilities.Utilities;
import org.objectweb.asm.ClassReader;
import org.objectweb.asm.ClassWriter;

import java.io.ByteArrayInputStream;
import java.io.InputStream;
import java.util.HashMap;
import java.util.List;
import java.util.Map;
import java.util.jar.JarInputStream;

public class ABICompiler {
    private String mainClassName;
    private byte[] mainClassBytes;
    private byte[] outputJarFile;
    private List<Method> callables;
    private Map<String, byte[]> classMap = new HashMap<>();
    private boolean stripLineNumber;

    public static ABICompiler compileJar(InputStream byteReader, boolean stripLineNumber) {
        return initCompilerAndCompile(byteReader, stripLineNumber);
    }

    public static ABICompiler compileJarBytes(byte[] rawBytes, boolean stripLineNumber) {
        return initCompilerAndCompile(new ByteArrayInputStream(rawBytes), stripLineNumber);
    }

    private static ABICompiler initCompilerAndCompile(InputStream byteReader, boolean stripLineNumber) {
        ABICompiler compiler = new ABICompiler();
        compiler.stripLineNumber = stripLineNumber;
        compiler.compile(byteReader);
        return compiler;
    }

    /**
     * We only want to expose the ABICompiler object once it is fully populated (_has_ compiled something) so we hide the constructor.
     * This can only be meaningfully called by our factory methods.
     */
    private ABICompiler() {
    }

    private void compile(InputStream byteReader) {
        try {
            safeLoadFromBytes(byteReader);
        } catch (Exception e) {
            e.printStackTrace();
        }

        ClassReader reader = new ClassReader(mainClassBytes);
        ClassWriter classWriter = new ClassWriter(ClassWriter.COMPUTE_MAXS);
        ABICompilerClassVisitor classVisitor = new ABICompilerClassVisitor(classWriter, stripLineNumber);
        reader.accept(classVisitor, 0);

        callables = classVisitor.getCallableInfo();
        mainClassBytes = classWriter.toByteArray();
        outputJarFile = JarBuilder.buildJarForExplicitClassNamesAndBytecode(mainClassName, mainClassBytes, this.classMap);
    }

    private void safeLoadFromBytes(InputStream byteReader) throws Exception {
        JarInputStream jarReader = new JarInputStream(byteReader, true);
        classMap = Utilities.extractClasses(jarReader, Utilities.NameStyle.DOT_NAME);
        mainClassName = Utilities.extractMainClassName(jarReader, Utilities.NameStyle.DOT_NAME);
        mainClassBytes = classMap.get(mainClassName);
        if (mainClassBytes == null) {
            throw new ABICompilerException("Cannot find main class: " + mainClassName);
        }
        classMap.remove(mainClassName);
    }

    public List<Method> getCallables() {
        return callables;
    }

    public byte[] getJarFileBytes() {
        return outputJarFile;
    }
}
