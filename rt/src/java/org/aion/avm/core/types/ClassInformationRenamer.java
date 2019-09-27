package org.aion.avm.core.types;

import i.RuntimeAssertionError;
import org.aion.avm.core.ClassRenamer;
import org.aion.avm.core.ClassRenamer.ArrayType;

/**
 * A static utility class that is used to rename a pre-rename {@link ClassInformation} object to a
 * post-rename {@link ClassInformation} object.
 */
public final class ClassInformationRenamer {

    /**
     * Returns classInformation but with itself and all of its super classes renamed.
     *
     * If any of the names are post-rename already, an exception will be thrown.
     *
     * @param classRenamer The class renamer utility to use.
     * @param classInformation The pre-rename class info.
     * @return the corresponding post-rename class info.
     */
    public static ClassInformation toPostRenameClassInfo(ClassRenamer classRenamer, ClassInformation classInformation) {
        RuntimeAssertionError.assertTrue(classInformation.isPreRenameClassInfo);
        RuntimeAssertionError.assertTrue(!classInformation.dotName.equals(CommonType.JAVA_LANG_OBJECT.dotName));

        String renamedSelf = classRenamer.toPostRenameOrRejectClass(classInformation.dotName, ArrayType.NOT_ARRAY);
        String renamedParent = getParentRenamed(classRenamer, classInformation);
        String[] renamedInterfaces = getInterfacesRenamed(classRenamer, classInformation);

        return ClassInformation.postRenameInfoFor(classInformation.isInterface, renamedSelf, renamedParent, renamedInterfaces);
    }

    /**
     * Returns the concrete super class of class info but renamed.
     *
     * If the given class info object has java.lang.Object as its super class and it is an interface,
     * then null will be returned (it must descend from IObject).
     */
    private static String getParentRenamed(ClassRenamer classRenamer, ClassInformation classInformation) {
        if (classInformation.superClassDotName == null) {
            return null;
        }

        if (classInformation.superClassDotName.equals(CommonType.JAVA_LANG_OBJECT.dotName)) {
            return (classInformation.isInterface) ? null : CommonType.SHADOW_OBJECT.dotName;
        }

        return classRenamer.toPostRenameOrRejectClass(classInformation.superClassDotName, ArrayType.NOT_ARRAY);
    }

    /**
     * Returns all of the interfaces of the given class info but renamed.
     */
    private static String[] getInterfacesRenamed(ClassRenamer classRenamer, ClassInformation classInformation) {
        String[] interfaces = classInformation.getInterfaces();

        String[] renamedInterfaces = new String[interfaces.length];
        for (int i = 0; i < interfaces.length; i++) {
            renamedInterfaces[i] = classRenamer.toPostRenameOrRejectClass(interfaces[i], ArrayType.NOT_ARRAY);
        }
        return renamedInterfaces;
    }

}
