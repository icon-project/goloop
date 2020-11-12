package org.aion.avm.tooling.deploy.eliminator;

import foundation.icon.ee.struct.Member;
import org.aion.avm.utilities.JarBuilder;
import org.aion.avm.utilities.Utilities;
import org.objectweb.asm.ClassReader;
import org.objectweb.asm.ClassWriter;

import java.io.ByteArrayInputStream;
import java.util.HashMap;
import java.util.List;
import java.util.Map;
import java.util.jar.JarInputStream;

public class UnreachableMethodRemover {

    public static byte[] optimize(byte[] jarBytes, Map<String, List<Member>> keptMethods) throws Exception {
        Map<String, byte[]> inputClassMap;
        Map<String, byte[]> outputClassMap = new HashMap<>();
        JarInputStream jarReader;

        jarReader = new JarInputStream(new ByteArrayInputStream(jarBytes), true);
        String mainClassName = Utilities.extractMainClassName(jarReader, Utilities.NameStyle.SLASH_NAME);
        inputClassMap = Utilities.extractClasses(jarReader, Utilities.NameStyle.SLASH_NAME);

        // Use the MethodReachabilityDetector to get the information about reachability
        Map<String, ClassInfo> classInfoMap = MethodReachabilityDetector.getClassInfoMap(mainClassName, inputClassMap, keptMethods);

        for (Map.Entry<String, byte[]> entry : inputClassMap.entrySet()) {
            ClassReader reader = new ClassReader(entry.getValue());
            ClassWriter classWriter = new ClassWriter(ClassWriter.COMPUTE_MAXS);
            ClassUnreachabilityVisitor classVisitor = new ClassUnreachabilityVisitor(classWriter,
                    classInfoMap.get(entry.getKey()).getMethodMap());
            reader.accept(classVisitor, 0);
            outputClassMap.put(Utilities.internalNameToFullyQualifiedName(entry.getKey()), classWriter.toByteArray());
        }

        byte[] mainClassBytes = outputClassMap.remove(Utilities.internalNameToFullyQualifiedName(mainClassName));
        return JarBuilder.buildJarForExplicitClassNamesAndBytecode(
                Utilities.internalNameToFullyQualifiedName(mainClassName), mainClassBytes, outputClassMap);
    }
}
