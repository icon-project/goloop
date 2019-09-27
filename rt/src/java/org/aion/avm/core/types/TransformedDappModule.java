package org.aion.avm.core.types;

import java.util.Map;

import i.PackageConstants;
import i.RuntimeAssertionError;


/**
 * Represents the DApp code once it has been validated and transformed but before it has been deployed and stored.
 * All fields are public since this object is effectively an immutable struct.
 * See issue-134 for more details on this design.
 */
public class TransformedDappModule {
    public static TransformedDappModule fromTransformedClasses(Map<String, byte[]> classes, String mainClass) {
        // We need to verify that both mainClass and the injected constant class are part of this set (note that the main class will NOT be renamed, in debug mode).
        RuntimeAssertionError.assertTrue(classes.containsKey(PackageConstants.kUserDotPrefix + mainClass) || classes.containsKey(mainClass));
        RuntimeAssertionError.assertTrue(classes.containsKey(PackageConstants.kConstantClassName));
        
        return new TransformedDappModule(classes, mainClass);
    }


    public final Map<String, byte[]> classes;
    public final String mainClass;

    private TransformedDappModule(Map<String, byte[]> classes, String mainClass) {
        this.classes = classes;
        this.mainClass = mainClass;
    }
}
