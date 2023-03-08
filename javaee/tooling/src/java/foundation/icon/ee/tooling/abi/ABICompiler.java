/*
 * Copyright 2019 ICON Foundation
 * Copyright (c) 2018 Aion Foundation https://aion.network/
 */

package foundation.icon.ee.tooling.abi;

import foundation.icon.ee.struct.Member;
import foundation.icon.ee.struct.PropertyMember;
import foundation.icon.ee.struct.StructDB;
import foundation.icon.ee.types.Method;
import foundation.icon.ee.util.ASM;
import foundation.icon.ee.util.Multimap;
import org.aion.avm.tooling.deploy.eliminator.ParameterNameRemover;
import org.aion.avm.utilities.JarBuilder;
import org.aion.avm.utilities.Utilities;
import org.objectweb.asm.ClassWriter;

import java.io.ByteArrayInputStream;
import java.io.InputStream;
import java.util.HashMap;
import java.util.HashSet;
import java.util.List;
import java.util.Map;
import java.util.Set;
import java.util.jar.JarInputStream;

public class ABICompiler {
    private String mainClassName;
    private byte[] mainClassBytes;
    private byte[] outputJarFile;
    private List<Method> callables;
    private Map<String, byte[]> classMap = new HashMap<>();
    private boolean stripLineNumber;
    private final Set<String> rootClasses = new HashSet<>();

    // slash name
    private final Map<String, List<Member>> keptMethods = new HashMap<>();
    private final Map<String, List<Member>> keptFields = new HashMap<>();

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

    private static void addKeptProperty(Map<String, List<Member>> map,
            PropertyMember p) {
        Multimap.add(map, p.getDeclaringType().getInternalName(),
                p.getMember());
    }

    private void collectKeptProperties(List<PropertyMember> props) {
        for (var p : props) {
            switch(p.getSort()) {
                case PropertyMember.FIELD:
                    addKeptProperty(keptFields, p);
                    break;
                case PropertyMember.GETTER:
                case PropertyMember.SETTER:
                    addKeptProperty(keptMethods, p);
                    break;
                default:
                    assert false;
            }
        }
    }

    private void compile(InputStream byteReader) {
        try {
            safeLoadFromBytes(byteReader);
        } catch (Exception e) {
            e.printStackTrace();
        }

        var structDB = new StructDB(classMap, true);
        var cw = new ClassWriter(ClassWriter.COMPUTE_MAXS);
        var cv = new ABICompilerClassVisitor(cw, classMap, structDB,
                stripLineNumber);
        ASM.accept(mainClassBytes, cv);

        callables = cv.getCallableInfo();
        mainClassBytes = cw.toByteArray();
        classMap.put(mainClassName, mainClassBytes);

        rootClasses.add(mainClassName);
        var internalName = Utilities.fullyQualifiedNameToInternalName(
                mainClassName);
        for (var m : callables) {
            Multimap.add(keptMethods, internalName,
                    new Member(m.getName(), m.getDescriptor()));
        }

        var paramStructs = structDB.getParameterStructs();
        for (var s : paramStructs) {
            rootClasses.add(s.getClassName());
            Multimap.add(keptMethods, s.getInternalName(),
                    new Member("<init>", "()V"));
            collectKeptProperties(structDB.getWritableProperties(s));
        }
        var returnStructs = structDB.getReturnStructs();
        for (var s : returnStructs) {
            rootClasses.add(s.getClassName());
            collectKeptProperties(structDB.getReadableProperties(s));
        }
        for (var e : classMap.entrySet()) {
            var icw = new ClassWriter(ClassWriter.COMPUTE_MAXS);
            var icv = ASM.accept(e.getValue(),
                    new KeptMemberCollector(new ParameterNameRemover(icw)));
            var name = Utilities.fullyQualifiedNameToInternalName(
                    e.getKey());
            Multimap.addAll(keptMethods, name, icv.getKeptMethods());
            Multimap.addAll(keptFields, name, icv.getKeptFields());
            classMap.put(e.getKey(), icw.toByteArray());
        }

        outputJarFile = JarBuilder.buildJarForExplicitClassNamesAndBytecode(
                mainClassName, classMap);
    }

    private void safeLoadFromBytes(InputStream byteReader) throws Exception {
        JarInputStream jarReader = new JarInputStream(byteReader, true);
        classMap = Utilities.extractClasses(jarReader, Utilities.NameStyle.DOT_NAME);
        mainClassName = Utilities.extractMainClassName(jarReader, Utilities.NameStyle.DOT_NAME);
        mainClassBytes = classMap.get(mainClassName);
        if (mainClassBytes == null) {
            throw new ABICompilerException("Cannot find main class: " + mainClassName);
        }
    }

    public List<Method> getCallables() {
        return callables;
    }

    public byte[] getJarFileBytes() {
        return outputJarFile;
    }

    public Set<String> getRootClasses() {
        return rootClasses;
    }

    public Map<String, List<Member>> getKeptMethods() {
        return keptMethods;
    }

    public Map<String, List<Member>> getKeptFields() {
        return keptFields;
    }
}
