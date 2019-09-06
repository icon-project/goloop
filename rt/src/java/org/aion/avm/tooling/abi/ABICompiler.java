package org.aion.avm.tooling.abi;

import java.io.ByteArrayInputStream;
import java.io.DataOutputStream;
import java.io.FileInputStream;
import java.io.FileNotFoundException;
import java.io.FileOutputStream;
import java.io.IOException;
import java.io.InputStream;
import java.io.OutputStream;
import java.io.PrintStream;
import java.util.Arrays;

import org.aion.avm.tooling.util.JarBuilder;
import org.aion.avm.tooling.util.Utilities;
import org.aion.avm.userlib.*;
import org.aion.avm.userlib.abi.*;
import org.objectweb.asm.ClassReader;
import org.objectweb.asm.ClassWriter;

import java.util.ArrayList;
import java.util.HashMap;
import java.util.List;
import java.util.Map;
import java.util.jar.JarInputStream;

import org.objectweb.asm.Type;

public class ABICompiler {

    private static final int DEFAULT_VERSION_NUMBER = 0;
    private static Class[] requiredUserlibClasses = new Class[] {ABIDecoder.class, ABIEncoder.class,
        ABIStreamingEncoder.class, ABIException.class, ABIToken.class, AionBuffer.class, AionList.class, AionMap.class, AionSet.class, AionUtilities.class};

    private String mainClassName;
    private byte[] mainClassBytes;
    private byte[] outputJarFile;
    private List<String> callables = new ArrayList<>();
    private List<Type> initializables = new ArrayList<>();
    private Map<String, byte[]> classMap = new HashMap<>();

    public static void main(String[] args) {
        if (args.length < 1 || args.length > 2) {
            System.out.println("Invalid parameters!");
            usage();
            System.exit(1);
        }

        String jarPath = args[0];
        int version;
        if (args.length > 1) {
            version = Integer.valueOf(args[1]);
        } else {
            version = DEFAULT_VERSION_NUMBER;
        }

        ABICompiler compiler = new ABICompiler();
        FileInputStream fileInputStream = null;
        try {
            fileInputStream = new FileInputStream(jarPath);
        } catch (FileNotFoundException e) {
            e.printStackTrace();
            System.exit(1);
        }

        compiler.compile(fileInputStream, version);

        compiler.writeAbi(System.out, version);

        try {
            DataOutputStream dout =
                    new DataOutputStream(new FileOutputStream("outputJar" + ".jar"));
            dout.write(compiler.getJarFileBytes());
            dout.close();
        } catch (IOException e) {
            e.printStackTrace();
        }
    }

    private static void usage() {
        System.out.println("Usage: ABICompiler <DApp jar path> <abi version number>");
    }


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
        ABICompilerClassVisitor classVisitor = new ABICompilerClassVisitor(classWriter, version) {};
        reader.accept(classVisitor, 0);

        callables = classVisitor.getCallableSignatures();
        initializables = classVisitor.getInitializableTypes();
        mainClassBytes = classWriter.toByteArray();


        Class<?>[] missingUserlib = getMissingUserlibClasses(this.classMap);
        outputJarFile = JarBuilder.buildJarForExplicitClassNamesAndBytecode(mainClassName, mainClassBytes, this.classMap, missingUserlib); 
    }

    public void writeAbi(OutputStream rawStream, int version) {
        // We want this to know about new lines so use a PrintStream.
        PrintStream abiStream = new PrintStream(rawStream);
        // This is to stay compatible with previous abi generated files that used a float to represent the version number
        abiStream.println(getVersionNumberForABIFilePrint(version));
        abiStream.println(this.mainClassName);

        abiStream.print("Clinit: (");
        int numberOfInitializables = this.initializables.size();
        if (numberOfInitializables > 0) {
            for (int i = 0; i < numberOfInitializables - 1; i++) {
                abiStream.print(
                    ABIUtils.shortenClassName(this.initializables.get(i).getClassName()) + ", ");
            }
            abiStream.print(ABIUtils
                .shortenClassName(this.initializables.get(numberOfInitializables - 1).getClassName()));
        }
        abiStream.print(")");

        abiStream.println();
        for (String s : this.callables) {
            abiStream.println(s);
        }
    }

    private void safeLoadFromBytes(InputStream byteReader) throws Exception {
        JarInputStream jarReader = new JarInputStream(byteReader, true);
        classMap = Utilities.extractClasses(jarReader, Utilities.NameStyle.DOT_NAME);
        mainClassName = Utilities.extractMainClassName(jarReader, Utilities.NameStyle.DOT_NAME);
        mainClassBytes = classMap.get(mainClassName);
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

    public List<String> getCallables() {
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

    public static int getDefaultVersionNumber() {
        return DEFAULT_VERSION_NUMBER;
    }

    public static String getVersionNumberForABIFilePrint(int version) {
        return (version == 0) ? "0.0" : String.valueOf(version);
    }

    public byte[] getJarFileBytes() {
        return outputJarFile;
    }
}
