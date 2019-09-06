package org.aion.avm.tooling.deploy.eliminator;

import java.io.ByteArrayInputStream;
import java.util.HashMap;
import java.util.Map;
import java.util.jar.JarInputStream;

import org.aion.avm.tooling.util.JarBuilder;
import org.aion.avm.tooling.util.Utilities;
import org.objectweb.asm.ClassReader;
import org.objectweb.asm.ClassWriter;

public class UnreachableMethodRemover {

    public static byte[] optimize(byte[] jarBytes) throws Exception {

        Map<String, byte[]> inputClassMap;
        Map<String, byte[]> outputClassMap = new HashMap<>();

        JarInputStream jarReader;

        jarReader = new JarInputStream(new ByteArrayInputStream(jarBytes), true);
        String mainClassName = Utilities.extractMainClassName(jarReader, Utilities.NameStyle.SLASH_NAME);
        inputClassMap = Utilities.extractClasses(jarReader, Utilities.NameStyle.SLASH_NAME);

        // Use the MethodReachabilityDetector to get the information about reachability
        Map<String, ClassInfo> classInfoMap = MethodReachabilityDetector.getClassInfoMap(mainClassName, inputClassMap);

        for (Map.Entry<String, byte[]> entry : inputClassMap.entrySet()) {
            ClassReader reader = new ClassReader(entry.getValue());
            ClassWriter classWriter = new ClassWriter(ClassWriter.COMPUTE_MAXS);
            ClassUnreachabilityVisitor classVisitor = new ClassUnreachabilityVisitor(classWriter,
                classInfoMap.get(entry.getKey()).getMethodMap());
            reader.accept(classVisitor, 0);
            outputClassMap.put(Utilities.internalNameToFulllyQualifiedName(entry.getKey()), classWriter.toByteArray());
        }

        byte[] mainClassBytes = outputClassMap.remove(Utilities.internalNameToFulllyQualifiedName(mainClassName));
        return JarBuilder.buildJarForExplicitClassNamesAndBytecode(Utilities.internalNameToFulllyQualifiedName(mainClassName), mainClassBytes, outputClassMap);
    }

}

