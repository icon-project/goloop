package org.aion.avm.core.verification;

import java.util.Map;

import i.UncaughtException;


/**
 * The class which provides the high-level helpers for our pre-transformation class verification.
 * This, internally, loads classes without initialization, hence invoking the JVM's verifier but not calling &lt;clinit%gt;.
 */
public class Verifier {
    /**
     * Verifies the untrusted classes by loading them, without invoking &lt;clinit%gt;.
     * 
     * @param classes The map of class names (dot-style) to class bytecode which should be verified.
     * @throws UncaughtException Thrown when something goes wrong during verification (generally a required class not being found).
     */
    public static void verifyUntrustedClasses(Map<String, byte[]> classes) {
        try {
            internalVerifyUntrustedClasses(classes);
        } catch (Throwable t) {
            throw new UncaughtException(t);
        }
    }

    private static void internalVerifyUntrustedClasses(Map<String, byte[]> classes) throws Throwable {
        VerifierClassLoader loader = new VerifierClassLoader(classes);
        
        // Ask that loader to load each class.
        for (String name : classes.keySet()) {
            // We don't want to initialize since this seems to be what causes <clinit> to run.
            boolean initialize = false;
            Class.forName(name, initialize, loader);
        }
        
        // Verify that each one was loaded.
        boolean isEmpty = (0 == loader.getNotYetLoadedCount());
        // (this can't logically happen - it is just here to make sure a future change doesn't break something).
        if (!isEmpty) {
            throw new AssertionError("Not all pre-transform classes were loaded");
        }
    }
}
