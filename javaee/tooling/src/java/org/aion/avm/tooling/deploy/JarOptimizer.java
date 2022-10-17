package org.aion.avm.tooling.deploy;

import org.aion.avm.utilities.JarBuilder;
import org.aion.avm.utilities.Utilities;
import org.objectweb.asm.ClassReader;
import org.objectweb.asm.ClassVisitor;
import org.objectweb.asm.ClassWriter;
import org.objectweb.asm.Opcodes;
import org.objectweb.asm.signature.SignatureVisitor;

import java.io.ByteArrayInputStream;
import java.io.DataOutputStream;
import java.io.FileInputStream;
import java.io.FileOutputStream;
import java.io.IOException;
import java.util.Collection;
import java.util.HashSet;
import java.util.Map;
import java.util.Set;
import java.util.jar.JarInputStream;

public class JarOptimizer {

    private static final boolean loggingEnabled = false;
    private final boolean preserveDebugInfo;

    public static void main(String[] args) {
        if (args.length != 1) {
            System.err.println("Input the path to the jar file.");
            System.exit(0);
        }

        //remove debug information when the tool is run
        JarOptimizer optimizer = new JarOptimizer(false);

        try (FileInputStream fileInputStream = new FileInputStream(args[0])) {
            byte[] optimizedJarBytes = optimizer.optimize(fileInputStream.readAllBytes());

            int pathLength = args[0].lastIndexOf("/") + 1;
            String outputJarName = args[0].substring(0, pathLength) + "minimized_" + args[0].substring(pathLength);
            optimizer.writeOptimizedJar(outputJarName, optimizedJarBytes);

        } catch (IOException e) {
            e.printStackTrace();
            System.exit(0);
        }
    }

    public JarOptimizer(boolean preserveDebugInfo) {
        this.preserveDebugInfo = preserveDebugInfo;
    }

    public byte[] optimize(byte[] jarBytes) {
        Map<String, byte[]> classMap;
        Set<String> visitedClasses = new HashSet<>();

        try {
            JarInputStream jarReader = new JarInputStream(new ByteArrayInputStream(jarBytes), true);
            String mainClassName = Utilities.extractMainClassName(jarReader, Utilities.NameStyle.DOT_NAME);
            classMap = Utilities.extractClasses(jarReader, Utilities.NameStyle.DOT_NAME);

            traverse(mainClassName, visitedClasses, classMap);

            return buildOptimizedJar(visitedClasses, classMap, mainClassName);
        } catch (IOException e) {
            e.printStackTrace();
            throw new RuntimeException(e);
        }
    }

    public byte[] optimize(byte[] jarBytes, Collection<String> rootClasses) {
        Map<String, byte[]> classMap;
        Set<String> visitedClasses = new HashSet<>();

        try {
            JarInputStream jarReader = new JarInputStream(new ByteArrayInputStream(jarBytes), true);
            String mainClassName = Utilities.extractMainClassName(jarReader, Utilities.NameStyle.DOT_NAME);
            classMap = Utilities.extractClasses(jarReader, Utilities.NameStyle.DOT_NAME);

            for (var c : rootClasses) {
                traverse(c, visitedClasses, classMap);
            }

            return buildOptimizedJar(visitedClasses, classMap, mainClassName);
        } catch (IOException e) {
            e.printStackTrace();
            throw new RuntimeException(e);
        }
    }

    private void traverse(String className, Set<String> visitedClasses, Map<String, byte[]> classMap) {
        visitedClasses.add(className);
        Set<String> referencedClasses = visitClass(className, classMap);

        if (loggingEnabled) {
            System.out.println("visited " + className);
            for (String c : referencedClasses) {
                System.out.println("  referenced " + c);
            }
        }

        for (String c : referencedClasses) {
            if (classMap.containsKey(c) && !visitedClasses.contains(c)) {
                traverse(c, visitedClasses, classMap);
            }
        }
    }

    private Set<String> visitClass(String className, Map<String, byte[]> classMap) {

        DependencyCollector dependencyCollector = new DependencyCollector();

        ClassReader reader = new ClassReader(classMap.get(className));

        SignatureVisitor signatureVisitor = new SignatureDependencyVisitor(dependencyCollector);
        ClassWriter writer = new ClassWriter(ClassWriter.COMPUTE_MAXS);
        ClassVisitor classVisitor = new ClassDependencyVisitor(signatureVisitor, dependencyCollector, writer, preserveDebugInfo);
        reader.accept(classVisitor, 0);

        classMap.put(className, writer.toByteArray());
        return dependencyCollector.getDependencies();
    }

    private byte[] buildOptimizedJar(Set<String> visitedClasses, Map<String, byte[]> classMap, String mainClassName) {
        if (loggingEnabled) {
            System.out.println("Need to remove " + (classMap.entrySet().size() - visitedClasses.size()) + " out of " + classMap.entrySet().size() + " classes.");
            classMap.forEach((key, value) -> {
                if (!visitedClasses.contains(key)) {
                    System.out.println(" - " + key);
                }
            });
        }

        classMap.entrySet().removeIf(e -> !visitedClasses.contains(e.getKey()));

        // update outer class bytes of removed inner classes
        for (String className : visitedClasses) {
            ClassReader reader = new ClassReader(classMap.get(className));
            ClassWriter writer = new ClassWriter(ClassWriter.COMPUTE_MAXS);

            ClassVisitor classVisitor = new ClassVisitor(Opcodes.ASM7, writer) {
                @Override
                public void visitNestHost(String nestHost) {
                    if (visitedClasses.contains(Utilities.internalNameToFullyQualifiedName(nestHost))) {
                        super.visitNestHost(nestHost);
                    }
                }

                @Override
                public void visitNestMember(String nestMember) {
                    if (visitedClasses.contains(Utilities.internalNameToFullyQualifiedName(nestMember))) {
                        super.visitNestMember(nestMember);
                    }
                }

                @Override
                public void visitInnerClass(String name, String outerName, String innerName, int access) {
                    // Remove InnerClasses attributes
                }
            };
            reader.accept(classVisitor, 0);
            classMap.replace(className, writer.toByteArray());
        }

        assertTrue(classMap.entrySet().size() == visitedClasses.size());

        byte[] mainClassBytes = classMap.get(mainClassName);
        classMap.remove(mainClassName, mainClassBytes);
        return JarBuilder.buildJarForExplicitClassNamesAndBytecode(mainClassName, mainClassBytes, classMap);
    }

    private void writeOptimizedJar(String jarName, byte[] jarBytes) {
        try (DataOutputStream dout = new DataOutputStream(new FileOutputStream(jarName))) {
            dout.write(jarBytes);
        } catch (IOException e) {
            e.printStackTrace();
        }

        System.out.println("Successfully created jar. \n" + jarName);
    }

    private static void assertTrue(boolean flag) {
        // We use a private helper to manage the assertions since the JDK default disables them.
        if (!flag) {
            throw new AssertionError("Case must be true");
        }
    }
}
