/*
 * Copyright 2019 ICON Foundation
 * Copyright (c) 2018 Aion Foundation https://aion.network/
 */

package foundation.icon.ee.tooling.abi;

import foundation.icon.ee.types.Method;
import org.aion.avm.tooling.util.JarBuilder;
import org.aion.avm.tooling.util.Utilities;
import org.aion.avm.userlib.*;
import org.aion.avm.userlib.abi.*;
import org.objectweb.asm.ClassReader;
import org.objectweb.asm.ClassWriter;

import java.io.ByteArrayInputStream;
import java.io.InputStream;
import java.util.ArrayList;
import java.util.Arrays;
import java.util.HashMap;
import java.util.List;
import java.util.Map;
import java.util.jar.JarInputStream;

public class ABICompiler {

    private static final int DEFAULT_VERSION_NUMBER = 1;
    private static Class[] requiredUserlibClasses = new Class[] {
            ABIDecoder.class, ABIEncoder.class, ABIStreamingEncoder.class, ABIException.class, ABIToken.class,
            AionBuffer.class, AionList.class, AionMap.class, AionSet.class, AionUtilities.class};

    private String mainClassName;
    private byte[] mainClassBytes;
    private byte[] outputJarFile;
    private List<Method> callables;
    private Map<String, byte[]> classMap = new HashMap<>();

    public static ABICompiler compileJar(InputStream byteReader) {
        return initCompilerAndCompile(byteReader, DEFAULT_VERSION_NUMBER);
    }

    public static ABICompiler compileJar(InputStream byteReader, int version) {
        return initCompilerAndCompile(byteReader, version);
    }

    public static ABICompiler compileJarBytes(byte[] rawBytes) {
        return initCompilerAndCompile(new ByteArrayInputStream(rawBytes), DEFAULT_VERSION_NUMBER);
    }

    public static ABICompiler compileJarBytes(byte[] rawBytes, int version) {
        return initCompilerAndCompile(new ByteArrayInputStream(rawBytes), version);
    }

    private static ABICompiler initCompilerAndCompile(InputStream byteReader, int version) {
        ABICompiler compiler = new ABICompiler();
        compiler.compile(byteReader, version);
        return compiler;
    }

    /**
     * We only want to expose the ABICompiler object once it is fully populated (_has_ compiled something) so we hide the constructor.
     * This can only be meaningfully called by our factory methods.
     */
    private ABICompiler() {
    }

    private void compile(InputStream byteReader, int version) {
        try {
            safeLoadFromBytes(byteReader);
        } catch (Exception e) {
            e.printStackTrace();
        }

        ClassReader reader = new ClassReader(mainClassBytes);
        ClassWriter classWriter = new ClassWriter(ClassWriter.COMPUTE_MAXS);
        ABICompilerClassVisitor classVisitor = new ABICompilerClassVisitor(classWriter, version);
        reader.accept(classVisitor, 0);

        callables = classVisitor.getCallableInfo();
        mainClassBytes = classWriter.toByteArray();

        Class<?>[] missingUserlib = getMissingUserlibClasses(this.classMap);
        outputJarFile = JarBuilder.buildJarForExplicitClassNamesAndBytecode(mainClassName, mainClassBytes, this.classMap, missingUserlib);
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

    // This is public only because some tests use it to verify behaviour.
    public static Class<?>[] getMissingUserlibClasses(Map<String, byte[]> originalClassMap) {
        List<Class> classesToAdd = new ArrayList<>();
        for (Class clazz: requiredUserlibClasses) {
            String fullyQualifiedName = clazz.getName();
            String internalName = Utilities.fulllyQualifiedNameToInternalName(fullyQualifiedName);
            byte[] expectedBytes = Utilities.loadRequiredResourceAsBytes(internalName + ".class");

            if (originalClassMap.containsKey(fullyQualifiedName)) {
                if (!Arrays.equals(expectedBytes, originalClassMap.get(fullyQualifiedName))) {
                    throw new ABICompilerException("Input jar contains class " + fullyQualifiedName + " but does not have expect contents");
                }
            } else {
                classesToAdd.add(clazz);
            }
        }
        return classesToAdd.toArray(new Class[0]);
    }

    public List<Method> getCallables() {
        return callables;
    }

    public byte[] getMainClassBytes() {
        return mainClassBytes;
    }

    public String getMainClassName() {
        return mainClassName;
    }

    public Map<String, byte[]> getClassMap() {
        return classMap;
    }

    public byte[] getJarFileBytes() {
        return outputJarFile;
    }
}
