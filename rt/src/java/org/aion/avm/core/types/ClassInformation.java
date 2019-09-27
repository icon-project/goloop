package org.aion.avm.core.types;

import java.util.Arrays;
import i.RuntimeAssertionError;

/**
 * Information pertaining to a class. In particular, the name of the class as well as the names of
 * its super class and super interfaces, if it has any, as well as whether or not the class is an
 * interface.
 *
 * Class information can pertain to pre- and post-rename classes, and each class information object
 * is marked accordingly.
 *
 * Marking a class pre- or post-rename has no effect on any of the information (except that
 * post-rename classes with no super classes specified will be given either shadow Object or
 * IObject, to help with correctness) but is simply a correctness measure so that the user can track
 * whether or not they are holding a pre- or post-rename class. Additionally, pre-rename class
 * information can easily be converted into post-rename class information using the
 * {@link ClassInformationRenamer} class.
 *
 * A class information object should be used to hold information about exception wrappers or about
 * generated array wrappers (handwritten array wrappers are fine). Or simply: any generated wrapper
 * type.
 *
 * Note that java.lang.Object cannot be renamed because it is a proper post-rename type (and renaming
 * it is ambiguous anyway because of the shadow Object and IObject split).
 *
 * All naming is dot-style.
 *
 * This class is immutable.
 */
public final class ClassInformation {
    public final String superClassDotName;
    public final String dotName;
    public final boolean isInterface;
    public final boolean isPreRenameClassInfo;
    private final String[] superInterfacesDotNames; // never null - is empty array instead

    private ClassInformation(boolean isPreRenameClass, boolean isInterface, String self, String parent, String[] interfaces) {
        if (self == null) {
            throw new NullPointerException("Cannot construct class info with null self.");
        }

        // All classes except java.lang.Object must have at least one super class defined. This is a
        // cautionary measure to help ensure that no nodes in the hierarchy ever get orphaned.
        if (!self.equals(CommonType.JAVA_LANG_OBJECT.dotName)) {

            if ((!isPreRenameClass) && (parent == null) && ((interfaces == null) || (interfaces.length == 0))) {
                throw new IllegalArgumentException("Cannot construct class info with no super classes defined: " + self);
            }

        }

        // Ensure that the self and parent names are dot-style (we check interfaces below).
        RuntimeAssertionError.assertTrue(!self.contains("/"));
        RuntimeAssertionError.assertTrue((parent == null) || (!parent.contains("/")));

        this.isPreRenameClassInfo = isPreRenameClass;
        this.isInterface = isInterface;
        this.dotName = self;
        this.superClassDotName = parent;
        this.superInterfacesDotNames = (interfaces == null) ? new String[0] : new String[interfaces.length];

        // Make a copy of interfaces, ensure each is dot-style in the process.
        if (interfaces != null) {
            for (int i = 0; i < interfaces.length; i++) {
                RuntimeAssertionError.assertTrue((interfaces[i] != null) && (!interfaces[i].contains("/")));
                this.superInterfacesDotNames[i] = interfaces[i];
            }
        }
    }

    public static ClassInformation postRenameInfofrom(CommonType specialType) {
        if (specialType == null) {
            throw new NullPointerException("Cannot create class info from a null special type.");
        }

        return new ClassInformation(false, specialType.isInterface, specialType.dotName, specialType.superClassDotName, specialType.superInterfacesDotNames);
    }

    public static ClassInformation preRenameInfofrom(CommonType specialType) {
        if (specialType == null) {
            throw new NullPointerException("Cannot create class info from a null special type.");
        }

        return new ClassInformation(true, specialType.isInterface, specialType.dotName, specialType.superClassDotName, specialType.superInterfacesDotNames);
    }

    public static ClassInformation preRenameInfoFor(boolean isInterface, String self, String parent, String[] interfaces) {
        return new ClassInformation(true, isInterface, self, parent, interfaces);
    }

    /**
     * Constructs a new class info for a post-rename class.
     *
     * No checks are performed to ensure that these are in fact post-rename class names because there
     * are legitimate cases for non-renamed classes to enter the post-rename landscape. Namely, if
     * debug mode is enabled then pre-rename user-defined classes will not be renamed.
     *
     * However, no post-rename class is allowed to have no super classes defined. If this is detected,
     * then the class will be reparented under IObject or shadow Object depending on whether or not
     * it is an interface.
     */
    public static ClassInformation postRenameInfoFor(boolean isInterface, String dotName, String superClassDotName, String[] superInterfacesDotNames) {
        String superClass = superClassDotName;
        String[] superInterfaces = superInterfacesDotNames;

        // Notice that slash names can obscure these checks. That is ok, the constructor rejects all
        // slash names, so we will catch the error there.
        if (isInterface) {

            // If the super class is java.lang.Object we remove this and ensure IObject is a super interface.
            if ((superClassDotName != null) && (superClassDotName.equals(CommonType.JAVA_LANG_OBJECT.dotName))) {
                superClass = null;
                superInterfaces = addIObjectIfAbsent(superInterfacesDotNames);
            }

            // If no super is defined at all, then IObject is the only super.
            if ((superClassDotName == null) && ((superInterfacesDotNames == null) || (superInterfacesDotNames.length == 0))) {
                superInterfaces = addIObjectIfAbsent(superInterfacesDotNames);
            }

        } else {

            // If no super is defined at all, then shadow Object is the only super.
            if ((superClassDotName == null) && ((superInterfacesDotNames == null) || (superInterfacesDotNames.length == 0))) {
                superClass = CommonType.SHADOW_OBJECT.dotName;
            }

        }

        return new ClassInformation(false, isInterface, dotName, superClass, superInterfaces);
    }

    /**
     * Returns an array of all super classes and super interfaces, if any.
     *
     * Returns an empty array if none.
     */
    public String[] superClasses() {
        if (this.superClassDotName == null) {
            return getInterfaces();
        }

        // This is safe since we know we never hold onto a null interface array.
        String[] interfaces = Arrays.copyOf(this.superInterfacesDotNames, this.superInterfacesDotNames.length + 1);
        interfaces[this.superInterfacesDotNames.length] = this.superClassDotName;
        return interfaces;
    }

    /**
     * Returns an array of all the super interfaces or an empty array if none.
     */
    public String[] getInterfaces() {
        return Arrays.copyOf(this.superInterfacesDotNames, this.superInterfacesDotNames.length);
    }

    public String rawString() {
        return "[" + (this.isInterface ? "interface" : "class") + "], "
            + "name = '" + this.dotName + "', "
            + "super class = '" + this.superClassDotName + "', "
            + "# of interfaces = " + this.superInterfacesDotNames.length;
    }

    /**
     * Returns an array containing only IObject if interfaces is null or empty.
     *
     * Otherwise, if interfaces is non-empty, returns the same array unless IObject is not already
     * in the array, in which case IObject is added to it.
     */
    private static String[] addIObjectIfAbsent(String[] interfaces) {
        if (interfaces == null) {
            return new String[]{ CommonType.I_OBJECT.dotName };
        }

        // If interfaces already contains IObject then we are done.
        for (String interfaceName : interfaces) {
            if (interfaceName.equals(CommonType.I_OBJECT.dotName)) {
                return interfaces;
            }
        }

        // Otherwise, we add IObject to the array.
        String[] interfacesWithIObject = Arrays.copyOf(interfaces, interfaces.length + 1);
        interfacesWithIObject[interfaces.length] = CommonType.I_OBJECT.dotName;
        return interfacesWithIObject;
    }

    @Override
    public String toString() {
        return "ClassInformation { " + rawString() + " }";
    }

    @Override
    public boolean equals(Object other) {
        if (!(other instanceof ClassInformation)) {
            return false;
        }

        ClassInformation otherInfo = (ClassInformation) other;

        boolean sameSuperClasses = (this.superClassDotName == null)
            ? otherInfo.superClassDotName == null
            : this.superClassDotName.equals(otherInfo.superClassDotName);

        return ((this.isInterface == otherInfo.isInterface)
            && (this.dotName.equals(otherInfo.dotName))
            && (sameSuperClasses)
            && (Arrays.equals(this.superInterfacesDotNames, otherInfo.superInterfacesDotNames)));
    }

    @Override
    public int hashCode() {
        int hash = 37;
        hash += this.isInterface ? 1 : 0;
        hash += this.dotName.hashCode();
        hash += this.superClassDotName == null ? 0 : this.superClassDotName.hashCode();
        return hash + Arrays.hashCode(this.superInterfacesDotNames);
    }

}
