package org.aion.avm.core;

import i.PackageConstants;


/**
 * A high-level abstraction over the questions related to "what classes can be referenced by this contract".
 * This encompasses 3 kinds of classes:
 * 1)  The JDK types we have shadowed.
 * 2)  The AVM runtime package.
 * 3)  The types defined within the user contract, itself.
 * 
 * NOTE:  This class is expected to be used only on post-renamed classes, only.
 * Additionally, this class does not consider debug mode.
 * Note that all predicates here are requested in terms of "slash-style" (aka "internal") class names.
 */
public class ClassWhiteList {
    /**
     * Checks if the class given is in any of our white-lists.
     * 
     * @param slashClassName The class to check.
     * @return True if we are allowed to access this class by any means we know.
     */
    public boolean isInWhiteList(String slashClassName) {
        return (slashClassName.startsWith(PackageConstants.kUserSlashPrefix)
                || slashClassName.startsWith(PackageConstants.kShadowSlashPrefix)
                || slashClassName.startsWith(PackageConstants.kShadowApiSlashPrefix)
                );
    }

    /**
     * Checks if the given class is in our JDK white-list.
     *
     * @param slashClassName The class to check.
     * @return True if we are allowed to access this class due to it being in our JDK white-list.
     */
    public boolean isJdkClass(String slashClassName) {
        return slashClassName.startsWith(PackageConstants.kShadowSlashPrefix);
    }
}
