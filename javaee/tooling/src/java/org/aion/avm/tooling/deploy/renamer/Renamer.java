package org.aion.avm.tooling.deploy.renamer;

import foundation.icon.ee.struct.Member;
import foundation.icon.ee.types.Method;
import foundation.icon.ee.util.Multimap;
import org.aion.avm.tooling.deploy.eliminator.ClassInfo;
import org.aion.avm.tooling.deploy.eliminator.MethodReachabilityDetector;
import org.aion.avm.utilities.JarBuilder;
import org.aion.avm.utilities.Utilities;
import org.objectweb.asm.ClassReader;
import org.objectweb.asm.ClassWriter;
import org.objectweb.asm.commons.ClassRemapper;
import org.objectweb.asm.commons.Remapper;
import org.objectweb.asm.commons.SimpleRemapper;
import org.objectweb.asm.tree.ClassNode;

import java.io.ByteArrayInputStream;
import java.io.DataOutputStream;
import java.io.FileInputStream;
import java.io.FileOutputStream;
import java.io.IOException;
import java.io.PrintStream;
import java.util.ArrayList;
import java.util.Arrays;
import java.util.Comparator;
import java.util.HashMap;
import java.util.LinkedHashMap;
import java.util.List;
import java.util.Map;
import java.util.jar.JarInputStream;
import java.util.stream.Collectors;

public class Renamer {

    public static void main(String[] args) {
        if (args.length < 1) {
            System.err.println("Input the path to the jar file.");
            System.exit(0);
        }
        String[] roots = Arrays.copyOfRange(args, 1, args.length);

        try (FileInputStream fileInputStream = new FileInputStream(args[0])) {
            byte[] renamedJarBytes = rename(fileInputStream.readAllBytes(), roots).getJarBytes();
            int pathLength = args[0].lastIndexOf("/") + 1;
            String outputJarName = args[0].substring(0, pathLength) + "renamed_" + args[0].substring(pathLength);
            writeOptimizedJar(outputJarName, renamedJarBytes);
        } catch (Exception e) {
            e.printStackTrace();
            System.exit(0);
        }
    }

    public static class Result {
        private byte[] jarBytes;
        private List<Method> callables;

        public Result(byte[] jarBytes, List<Method> callables) {
            this.jarBytes = jarBytes;
            this.callables = callables;
        }

        public byte[] getJarBytes() {
            return jarBytes;
        }

        public List<Method> getCallables() {
            return callables;
        }
    }

    public static Result rename(byte[] jarBytes, String[] roots) throws Exception {
        JarInputStream jarReader = new JarInputStream(new ByteArrayInputStream(jarBytes), true);
        String mainClassName = Utilities.extractMainClassName(jarReader, Utilities.NameStyle.DOT_NAME);
        jarReader.close();
        var l = Arrays.stream(roots).map(m -> {
            int idx = m.indexOf('(');
            return new Member(m.substring(0, idx), m.substring(idx));
        }).collect(Collectors.toList());
        var mmap = new HashMap<String, List<Member>>();
        mmap.put(mainClassName, l);
        var fmap = new HashMap<String, List<Member>>();
        return rename(jarBytes, null, mmap, fmap, null);
    }

    public static Result rename(byte[] jarBytes,
            List<Method> callables,
            Map<String, List<Member>> keptMethods,
            Map<String, List<Member>> keptFields,
            PrintStream log) throws Exception {
        JarInputStream jarReader = new JarInputStream(new ByteArrayInputStream(jarBytes), true);
        String mainClassName = Utilities.extractMainClassName(jarReader, Utilities.NameStyle.SLASH_NAME);
        Map<String, ClassNode> sortedClassMap = sortBasedOnInnerClassLevel(extractClasses(jarReader));

        String[] newMainNameBuf = new String[1];
        List<Method> outCallables = new ArrayList<>();
        Map<String, ClassNode> renamedNodes = renameClassNodes(sortedClassMap,
                mainClassName, callables, keptMethods, keptFields, log,
                newMainNameBuf, outCallables);

        Map<String, byte[]> classNameByteCodeMap = getClassBytes(renamedNodes);
        String newMainClassName = newMainNameBuf[0];
        byte[] mainClassBytes = classNameByteCodeMap.get(newMainClassName);
        classNameByteCodeMap.remove(newMainClassName, mainClassBytes);

        var outJarBytes = JarBuilder.buildJarForExplicitClassNamesAndBytecode(
                Utilities.internalNameToFullyQualifiedName(newMainClassName), mainClassBytes, classNameByteCodeMap);
        return new Result(outJarBytes, outCallables);
    }

    public static Map<String, ClassNode> sortBasedOnInnerClassLevel(Map<String, ClassNode> classMap) {
        Comparator<Map.Entry<String, ClassNode>> keyComparator =
                (n1, n2) -> Long.compare((n1.getKey().chars().filter(ch -> ch == '$').count()), (n2.getKey().chars().filter(ch -> ch == '$').count()));

        return classMap.entrySet().stream()
                .sorted(keyComparator)
                .collect(Collectors.toMap(Map.Entry::getKey, Map.Entry::getValue, (e1, e2) -> e1, LinkedHashMap::new));
    }

    private static void dumpMapping(Map<String, String> mappedNames,
            PrintStream log) {
        if (log != null) {
            mappedNames.forEach((k, v) -> log.format("%s -> %s%n", k, v));
        }
    }

    private static Map<String, List<Member>> remap(Map<String, List<Member>> in,
            Remapper remapper) {
        var out = new HashMap<String, List<Member>>();
        for (var e : in.entrySet()) {
            var newClass = remapper.mapType(e.getKey());
            var members = e.getValue();
            for (var m : members) {
                Multimap.add(out, newClass, new Member(
                        m.getName(),
                        remapper.mapDesc(m.getDescriptor())
                ));
            }
        }
        return out;
    }

    private static Map<String, ClassNode> renameClassNodes(
            Map<String, ClassNode> sortedClassMap, String mainClassName,
            List<Method> callables, Map<String, List<Member>> keptMethods,
            Map<String, List<Member>> keptFields, PrintStream log,
            String[] out_newMainName, List<Method> out_newCallables
    ) throws Exception {
        // rename classes
        Map<String, String> mappedNames = ClassRenamer.renameClasses(sortedClassMap);
        dumpMapping(mappedNames, log);
        if (out_newMainName != null && out_newMainName.length > 0) {
            out_newMainName[0] = mappedNames.get(mainClassName);
        }
        Map<String, ClassNode> newClassNameMap = applyMapping(sortedClassMap, mappedNames);
        var remapper = new SimpleRemapper(mappedNames);
        keptMethods = remap(keptMethods, remapper);
        keptFields = remap(keptFields, remapper);
        callables.stream()
                .map(m -> m.remap(remapper))
                .forEachOrdered(out_newCallables::add);

        // rename methods
        String newMainClassName = mappedNames.get(mainClassName);
        Map<String, ClassInfo> classInfoMap = MethodReachabilityDetector.getClassInfoMap(newMainClassName, getClassBytes(newClassNameMap), keptMethods);
        var roots = Multimap.getAllValues(keptMethods).stream()
                .map(Member::getMethodID).toArray(String[]::new);
        mappedNames = MethodRenamer.renameMethods(newClassNameMap, classInfoMap, newMainClassName, roots);
        dumpMapping(mappedNames, log);
        Map<String, ClassNode> newMethodNameMap = applyMapping(newClassNameMap, mappedNames);

        // rename fields
        mappedNames = FieldRenamer.renameFields(newMethodNameMap, classInfoMap,
                keptFields);
        dumpMapping(mappedNames, log);
        return applyMapping(newMethodNameMap, mappedNames);
    }

    private static Map<String, ClassNode> applyMapping(Map<String, ClassNode> classMap, Map<String, String> classNameMap) {
        SimpleRemapper remapper = new SimpleRemapper(classNameMap);
        Map<String, ClassNode> newClassMap = new HashMap<>();
        for (ClassNode node : classMap.values()) {
            ClassNode copy = new ClassNode();
            ClassRemapper adapter = new ClassRemapper(copy, remapper);
            node.accept(adapter);
            newClassMap.put(copy.name, copy);
        }
        return newClassMap;
    }

    public static Map<String, ClassNode> extractClasses(JarInputStream jarReader) throws IOException {
        Map<String, ClassNode> classMap = new HashMap<>();
        Map<String, byte[]> classByteMap = Utilities.extractClasses(jarReader, Utilities.NameStyle.SLASH_NAME);
        classByteMap.forEach((key, value) -> {
            ClassNode c = new ClassNode();
            new ClassReader(value).accept(c, 0);
            classMap.put(key, c);
        });
        return classMap;
    }

    private static Map<String, byte[]> getClassBytes(Map<String, ClassNode> classMap) {
        Map<String, byte[]> byteMap = new HashMap<>();
        for (ClassNode node : classMap.values()) {
            ClassWriter writer = new ClassWriter(0);
            node.accept(writer);
            byte[] classBytes = writer.toByteArray();
            byteMap.put(node.name, classBytes);
        }
        return byteMap;
    }

    private static void writeOptimizedJar(String jarName, byte[] jarBytes) {
        try (DataOutputStream dout = new DataOutputStream(new FileOutputStream(jarName))) {
            dout.write(jarBytes);
        } catch (IOException e) {
            e.printStackTrace();
        }
        System.out.println("Successfully created jar. \n" + jarName);
    }
}
