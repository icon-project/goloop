package org.aion.avm.core.types;

import java.util.HashSet;
import java.util.Map;
import java.util.Map.Entry;
import java.util.Set;
import org.aion.avm.core.dappreading.LoadedJar;
import i.RuntimeAssertionError;
import org.objectweb.asm.ClassReader;

/**
 * A factory for generating class information objects from bulk sources that typically do not have
 * simple conversion logic.
 */
public final class ClassInformationFactory {

    /**
     * Returns a set of all the class infos derived from the classes in the specified jar.
     *
     * Note that the returned classes are pre-rename classes since we are reading a pre-rename jar.
     *
     * @param jar The jar from which the classes are to be found.
     * @return The class info of each class defined in the jar.
     */
    public Set<ClassInformation> fromUserDefinedPreRenameJar(LoadedJar jar) {
        if (jar == null) {
            throw new NullPointerException("Cannot derive class information from a null jar.");
        }

        Set<ClassInformation> classInfos = new HashSet<>();

        Map<String, byte[]> classNameToBytes = jar.classBytesByQualifiedNames;
        for (Entry<String, byte[]> classNameToBytesEntry : classNameToBytes.entrySet()) {
            classInfos.add(fromClassBytes(classNameToBytesEntry.getValue(), false));
        }

        return classInfos;
    }

    /**
     * Returns a set of all class infos derived from the classes in the specified jar.
     *
     * The returned class infos are all marked as post-rename class infos since these classes come
     * from a post-rename jar.
     */
    public Set<ClassInformation> fromPostRenameJar(LoadedJar jar) {
        if (jar == null) {
            throw new NullPointerException("Cannot derive class information from a null jar.");
        }

        Set<ClassInformation> classInfos = new HashSet<>();

        Map<String, byte[]> classNameToBytes = jar.classBytesByQualifiedNames;
        for (Entry<String, byte[]> classNameToBytesEntry : classNameToBytes.entrySet()) {
            String className = classNameToBytesEntry.getKey();

            // Since we are constructing from a post-rename jar, we don't want to re-add any object types.
            if (!className.equals(CommonType.JAVA_LANG_OBJECT.dotName) && !className.equals(CommonType.I_OBJECT.dotName) && !className.equals(CommonType.SHADOW_OBJECT.dotName)) {
                classInfos.add(fromClassBytes(classNameToBytesEntry.getValue(), true));
            }
        }

        return classInfos;
    }

    /**
     * Returns a set of all class infos derived from the provided bytecodes.
     *
     * The returned class infos are all marked as pre-rename since they come from pre-rename
     * bytecode.
     */
    public Set<ClassInformation> fromPreRenameUserDefinedBytecode(Map<String, byte[]> classNameToBytecode, boolean preserveDebuggability) throws ClassNotFoundException {
        Set<ClassInformation> preRenameClassInfos = new HashSet<>();

        for (Map.Entry<String, byte[]> classNameToBytecodeEntry : classNameToBytecode.entrySet()) {
            RuntimeAssertionError.assertTrue(!classNameToBytecodeEntry.getKey().contains("/"));

            Class<?> loadedClass = this.getClass().getClassLoader().loadClass(classNameToBytecodeEntry.getKey());

            boolean isInterface = loadedClass.isInterface();
            String superClass = getConcreteSuperClass(loadedClass);
            String[] superInterfaces = getInterfaces(loadedClass);

            // If no supers are defined at all then we subclass under the appropriate Object type.
            if ((superClass == null) && ((superInterfaces == null) || (superInterfaces.length == 0))) {

                if (preserveDebuggability) {
                    if (isInterface) {
                        superInterfaces = new String[]{ CommonType.I_OBJECT.dotName };
                    } else {
                        superClass = CommonType.SHADOW_OBJECT.dotName;
                    }
                } else {
                    superClass = CommonType.JAVA_LANG_OBJECT.dotName;
                }

            }

            preRenameClassInfos.add(ClassInformation
                .preRenameInfoFor(isInterface, loadedClass.getName(), superClass, superInterfaces));
        }

        return preRenameClassInfos;
    }

    private String getConcreteSuperClass(Class<?> clazz) {
        Class<?> superClass = clazz.getSuperclass();
        return (superClass == null) ? null : superClass.getName();
    }

    private String[] getInterfaces(Class<?> clazz) {
        Class<?>[] interfacesAsClasses = clazz.getInterfaces();
        String[] interfaces = new String[interfacesAsClasses.length];

        for (int i = 0; i < interfacesAsClasses.length; i++) {
            interfaces[i] = interfacesAsClasses[i].getName();
        }

        return interfaces;
    }

    /**
     * Loads the class from the given bytes and produces a class info object for it.
     *
     * The class info object will be a pre-rename object if isRenamed is false. Otherwise, it will
     * be a post-rename object.
     *
     * No renaming will actually be done. It is simply that the class info will be marked correctly.
     */
    private ClassInformation fromClassBytes(byte[] classBytes, boolean isRenamed) {
        ClassReader reader = new ClassReader(classBytes);
        ClassInfoVisitor codeVisitor = new ClassInfoVisitor(isRenamed);
        reader.accept(codeVisitor, ClassReader.SKIP_FRAMES);
        return codeVisitor.getClassInfo();
    }

}
