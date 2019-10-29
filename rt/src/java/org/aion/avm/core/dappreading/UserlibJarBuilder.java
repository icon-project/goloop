package org.aion.avm.core.dappreading;

import java.util.Collections;
import java.util.Map;
import java.util.stream.Stream;

import org.aion.avm.userlib.*;
import org.aion.avm.userlib.abi.ABIDecoder;
import org.aion.avm.userlib.abi.ABIEncoder;
import org.aion.avm.userlib.abi.ABIException;
import org.aion.avm.userlib.abi.ABIStreamingEncoder;
import org.aion.avm.userlib.abi.ABIToken;
import org.aion.avm.utilities.JarBuilder;


/**
 * This just sits on top of the common JarBuilder, from the utilities module, stitching in the userlib where required by tests
 * in core and embed modules.
 */
public class UserlibJarBuilder {
    private static Class<?>[] userlibClasses = new Class[] {ABIDecoder.class, ABIEncoder.class,
        ABIStreamingEncoder.class, ABIException.class, ABIToken.class, AionBuffer.class, AionList.class, AionMap.class, AionSet.class, AionUtilities.class};

    /**
     * Creates the in-memory representation of a JAR with the given main class, other classes, and all classes in the Userlib.
     *
     * @param mainClass The main class to include and list in manifest (can be null).
     * @param otherClasses The other classes to include (main is already included).
     * @return The bytes representing this JAR.
     */
    public static byte[] buildJarForMainAndClassesAndUserlib(Class<?> mainClass, Class<?> ...otherClasses) {
        Class<?>[] combinedOtherClasses = Stream.of(otherClasses, userlibClasses).flatMap(Stream::of).toArray(Class<?>[]::new);
        return JarBuilder.buildJarForMainClassAndExplicitClassNamesAndBytecode(mainClass, Collections.emptyMap(), combinedOtherClasses);
    }

    /**
     * Creates the in-memory representation of a JAR with the given main class and other classes.
     * 
     * @param mainClass The main class to include and list in manifest (can be null).
     * @param otherClasses The other classes to include (main is already included).
     * @return The bytes representing this JAR.
     */
    public static byte[] buildJarForMainAndClasses(Class<?> mainClass, Class<?> ...otherClasses) {
        return JarBuilder.buildJarForMainClassAndExplicitClassNamesAndBytecode(mainClass, Collections.emptyMap(), otherClasses);
    }

    /**
     * Creates the in-memory representation of a JAR with the given classes and explicit main class name.
     * NOTE:  This method is really just used to build invalid JARs (main class might not be included).
     * 
     * @param mainClassName The name of the main class to reference in the manifest (cannot be null).
     * @param otherClasses The other classes to include (main is already included).
     * @return The bytes representing this JAR.
     */
    public static byte[] buildJarForExplicitMainAndClasses(String mainClassName, Class<?> ...otherClasses) {
        return JarBuilder.buildJarForExplicitMainAndClasses(mainClassName, otherClasses);
    }

    /**
     * Creates the in-memory representation of a JAR with the given class name and direct bytes.
     * NOTE:  This method is really just used to build invalid JARs (given classes may be corrupt/invalid).
     * 
     * @return The bytes representing this JAR.
     */
    public static byte[] buildJarForExplicitClassNameAndBytecode(String mainClassName, byte[] mainClassBytes) {
        return JarBuilder.buildJarForExplicitClassNamesAndBytecode(mainClassName, mainClassBytes, Collections.emptyMap());
    }

    /**
     * Creates the in-memory representation of a JAR with the given class names and direct bytes.
     * @return The bytes representing this JAR.
     */
    public static byte[] buildJarForExplicitClassNamesAndBytecode(String mainClassName, byte[] mainClassBytes, Map<String, byte[]> classMap, Class<?> ...otherClasses) {
        return JarBuilder.buildJarForExplicitClassNamesAndBytecode(mainClassName, mainClassBytes, classMap, otherClasses);
    }

    public static byte[] buildJarForExplicitClassNamesAndBytecodeAndUserlib(Class<?> mainClass, Map<String, byte[]> classMap, Class<?> ...otherClasses) {
        Class<?>[] combinedOtherClasses = Stream.of(otherClasses, userlibClasses).flatMap(Stream::of).toArray(Class<?>[]::new);
        return JarBuilder.buildJarForMainClassAndExplicitClassNamesAndBytecode(mainClass, classMap, combinedOtherClasses);
    }

    /**
     * Creates the in-memory representation of a JAR with the given class names and direct bytes, but a fixed main class.
     * @return The bytes representing this JAR.
     */
    public static byte[] buildJarForMainClassAndExplicitClassNamesAndBytecode(Class<?> mainClass, Map<String, byte[]> classMap) {
        return JarBuilder.buildJarForMainClassAndExplicitClassNamesAndBytecode(mainClass, classMap);
    }
}
