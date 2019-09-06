package org.aion.avm.tooling.util;

import java.io.ByteArrayOutputStream;
import java.io.IOException;
import java.util.HashSet;
import java.util.Map;
import java.util.Set;
import java.util.jar.Attributes;
import java.util.jar.JarEntry;
import java.util.jar.JarFile;
import java.util.jar.JarOutputStream;
import java.util.jar.Manifest;
import java.util.zip.ZipEntry;


/**
 * A utility to build in-memory JAR representations for tests and examples.
 * 
 * This is kept purely private and only the top-level factory method operates on the instances since they are stateful in ways which
 * would be complicated to communicate (streams are closed when reading the bytes, for example).
 */
public class JarBuilder {
    // AKI-135: We define this value as our fixed timestamp for tests (this number is similar to what we were using when the tests
    // were written and some of them assume compressed size).
    private static long FIXED_TIMESTAMP = 1_000_000_000_000L;


    /**
     * Creates the in-memory representation of a JAR with the given class names and direct bytes.
     * @return The bytes representing this JAR.
     */
    public static byte[] buildJarForExplicitClassNamesAndBytecode(String mainClassName, byte[] mainClassBytes, Map<String, byte[]> classMap, Class<?> ...otherClasses) {
        JarBuilder builder = new JarBuilder(null, mainClassName);
        try {
            builder.saveClassToStream(mainClassName, mainClassBytes);
            for (Map.Entry<String, byte[]> entry : classMap.entrySet()) {
                builder.saveClassToStream(entry.getKey(), entry.getValue());
            }
            for (Class<?> clazz : otherClasses) {
                builder.addClassAndInners(clazz);
            }
        } catch (IOException e) {
            // Can't happen - in-memory.
            throw new AssertionError(e);
        }
        return builder.toBytes();
    }

    public static byte[] buildJarForMainClassAndExplicitClassNamesAndBytecode(Class<?> mainClass, Map<String, byte[]> classMap, Class<?> ...otherClasses) {
        JarBuilder builder = new JarBuilder(mainClass, null);
        try {
            for (Map.Entry<String, byte[]> entry : classMap.entrySet()) {
                builder.saveClassToStream(entry.getKey(), entry.getValue());
            }
            for (Class<?> clazz : otherClasses) {
                builder.addClassAndInners(clazz);
            }
        } catch (IOException e) {
            // Can't happen - in-memory.
            throw new AssertionError(e);
        }
        return builder.toBytes();
    }

    private final ByteArrayOutputStream byteStream;
    private final JarOutputStream jarStream;
    private final Set<String> entriesInJar;

    private JarBuilder(Class<?> mainClass, String mainClassName) {
        // Build the manifest.
        Manifest manifest = new Manifest();
        Attributes mainAttributes = manifest.getMainAttributes();
        // Note that the manifest version seems to be required.  If it isn't specified, we don't see the main class.
        mainAttributes.put(Attributes.Name.MANIFEST_VERSION, "1.0");
        // The main class is technically optional (we mostly use a null main for testing cases).
        if (null != mainClass) {
            mainAttributes.put(Attributes.Name.MAIN_CLASS, mainClass.getName());
        } else if (null != mainClassName) {
            mainAttributes.put(Attributes.Name.MAIN_CLASS, mainClassName);
        }

        // Create the underlying byte stream (hold onto this for serialization).
        this.byteStream = new ByteArrayOutputStream();
        JarOutputStream stream = null;
        try {
            // We always write the manifest into the high-level JAR stream.
            stream = new JarOutputStream(this.byteStream);
            // AKI-135: We need to write the manifest, manually, to give it a deterministic timestamp.
            ZipEntry manifestEntry = new ZipEntry(JarFile.MANIFEST_NAME);
            manifestEntry.setTime(FIXED_TIMESTAMP);
            stream.putNextEntry(manifestEntry);
            manifest.write(stream);
            stream.closeEntry();
        } catch (IOException e) {
            // We are using a byte array so this can't happen.
            throw new AssertionError(e);
        }
        this.jarStream = stream;
        this.entriesInJar = new HashSet<>();

        // Finally, add this class.
        if (null != mainClass) {
            addClassAndInners(mainClass);
        }
    }

    /**
     * Loads the given class, any declared classes (named inner classes), and any anonymous inner classes.
     * 
     * @param clazz The class to load.
     * @return this, for easy chaining.
     */
    public JarBuilder addClassAndInners(Class<?> clazz) {
        try {
            // Load everything related to this class.
            loadClassAndAnonymous(clazz);
            // Now, include any declared classes.
            for (Class<?> one : clazz.getDeclaredClasses()) {
                addClassAndInners(one);
            }
        } catch (IOException e) {
            // We are serializing to a byte array so this is unexpected.
            throw new AssertionError(e);
        }
        return this;
    }

    private void loadClassAndAnonymous(Class<?> clazz) throws IOException {
        // Start with the fully-qualified class name, since we use that for addressing it.
        String className = clazz.getName();
        byte[] bytes = Utilities.loadRequiredResourceAsBytes(Utilities.fulllyQualifiedNameToInternalName(className) + ".class");
        if (null == bytes) {
            throw new AssertionError("Class bytes could not be found");
        }
        saveClassToStream(className, bytes);
        
        // Load any inner classes which might exist (these are just decimal suffixes, starting at 1.
        int i = 1;
        String innerName = className + "$" + Integer.toString(i);
        byte[] innerBytes = Utilities.loadRequiredResourceAsBytes(Utilities.fulllyQualifiedNameToInternalName(innerName) + ".class");
        while (null != innerBytes) {
            saveClassToStream(innerName, innerBytes);
            
            i += 1;
            innerName = className + "$" + Integer.toString(i);
            innerBytes = Utilities.loadRequiredResourceAsBytes(Utilities.fulllyQualifiedNameToInternalName(innerName) + ".class");
        }
    }

    private void saveClassToStream(String qualifiedClassName, byte[] bytes) throws IOException {
        // Convert this fully-qualified name into an internal name, since that is the serialized name it needs.
        String internalName = Utilities.fulllyQualifiedNameToInternalName(qualifiedClassName);
        if (this.entriesInJar.contains(internalName)) {
            // This is a static usage error.
            throw new AssertionError("Added class to JAR twice");
        }
        JarEntry entry = new JarEntry(internalName + ".class");
        // AKI-135: While we only use this utility in tests, it is still convenient if we force the timestamp for deterministic JAR creation.
        entry.setTime(FIXED_TIMESTAMP);
        this.jarStream.putNextEntry(entry);
        this.jarStream.write(bytes);
        this.jarStream.closeEntry();
        this.entriesInJar.add(internalName);
    }

    public byte[] toBytes() {
        try {
            this.jarStream.finish();
            this.jarStream.close();
            this.byteStream.close();
        } catch (IOException e) {
            // We are using a byte array so this can't happen.
            throw new AssertionError(e);
        }
        return this.byteStream.toByteArray();
    }
}
