package org.aion.avm.core.dappreading;

import org.aion.avm.utilities.Utilities;

import java.io.ByteArrayInputStream;
import java.io.IOException;
import java.util.Collections;
import java.util.HashMap;
import java.util.Map;
import java.util.jar.Attributes;
import java.util.jar.JarEntry;
import java.util.jar.JarInputStream;
import java.util.jar.Manifest;
import java.util.zip.ZipException;

/**
 * Converts the in-memory byte[] representation of a JAR into something we can easily interact with as an object.
 * Specifically, this involves asking for things like manifest data but, more commonly, the map of fully-qualified names to class file bytes.
 */
public class LoadedJar {
    // We probably want to put a concrete size limit on the uncompressed size of a class.
    // For now, we will use 1MiB, but this might change.
    private static final int MAX_CLASS_BYTES = 1024 * 1024;

    /**
     * Returns a representation of the JAR loaded from the given bytes,
     * or throws IOException if the JAR was malformed.
     * 
     * @param jar The in-memory JAR file.
     * @return The high-level JAR, or null if the input was malformed.
     */
    public static LoadedJar fromBytes(byte[] jar) throws IOException {
        try (ByteArrayInputStream byteReader = new ByteArrayInputStream(jar)) {
            return safeLoadFromBytes(byteReader);
        } catch (SecurityException e) {
            // This might happen if the JAR has a signature but it is invalid
            throw new IOException(e);
        } catch (SizeException e) {
            // This can happen if the JAR contains a single class which is larger than MAX_CLASS_BYTES
            throw new IOException(e);
        } catch (IOException e) {
            throw new ZipException(e.toString());
        }
    }

    private static LoadedJar safeLoadFromBytes(ByteArrayInputStream byteReader) throws IOException, SizeException {
        Map<String, byte[]> classBytesByQualifiedNames = new HashMap<>();
        String mainClassName = null;

        try (JarInputStream jarReader = new JarInputStream(byteReader, true)) {
            Manifest manifest = jarReader.getManifest();
            if (null != manifest) {
                Attributes mainAttributes = manifest.getMainAttributes();
                if (null != mainAttributes) {
                    mainClassName = mainAttributes.getValue(Attributes.Name.MAIN_CLASS);
                }
            }

            JarEntry entry = null;
            byte[] tempReadingBuffer = new byte[MAX_CLASS_BYTES];
            while (null != (entry = jarReader.getNextJarEntry())) {
                String name = entry.getName();
                // We already ready the manifest so now we only want to work on classes and not any of the special modularity ones.
                if (name.endsWith(".class")
                        && !name.equals("package-info.class")
                        && !name.equals("module-info.class")
                ) {
                    // replaceAll gives us the regex so we use "$".
                    String internalClassName = name.replaceAll(".class$", "");
                    String qualifiedClassName = Utilities.internalNameToFullyQualifiedName(internalClassName);
                    int readSize = jarReader.readNBytes(tempReadingBuffer, 0, tempReadingBuffer.length);
                    // Now, copy this part of the array as a correctly-sized classBytes.
                    byte[] classBytes = new byte[readSize];
                    if (0 != jarReader.available()) {
                        // This entry is too big.
                        throw new SizeException(name);
                    }
                    System.arraycopy(tempReadingBuffer, 0, classBytes, 0, readSize);
                    classBytesByQualifiedNames.put(qualifiedClassName, classBytes);
                }
            }
        }
        return new LoadedJar(classBytesByQualifiedNames, mainClassName);
    }

    public final Map<String, byte[]> classBytesByQualifiedNames;
    public final String mainClassName;

    public LoadedJar(Map<String, byte[]> classBytesByQualifiedNames, String mainClassName) {
        this.classBytesByQualifiedNames = Collections.unmodifiableMap(classBytesByQualifiedNames);
        this.mainClassName = mainClassName;
    }

    private static class SizeException extends Exception {
        private static final long serialVersionUID = 1L;
        public SizeException(String entryName) {
            super("Class file too big: " + entryName);
        }
    }
}
