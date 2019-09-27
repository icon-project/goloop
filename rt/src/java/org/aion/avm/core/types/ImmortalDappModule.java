package org.aion.avm.core.types;

import java.io.ByteArrayOutputStream;
import java.io.IOException;
import java.nio.file.attribute.FileTime;
import java.util.Map;
import java.util.jar.Attributes;
import java.util.jar.JarEntry;
import java.util.jar.JarFile;
import java.util.jar.JarOutputStream;
import java.util.jar.Manifest;

import java.util.zip.ZipEntry;
import org.aion.types.AionAddress;
import org.aion.avm.core.dappreading.LoadedJar;


/**
 * Represents the DApp code once it has been validated, transformed, and stripped of any code/data only required for the initial deployment call.
 * This is the form the module will take, in storage.
 * All fields are public since this object is effectively an immutable struct.
 * See issue-134 for more details on this design.
 */
public class ImmortalDappModule {
    // Note that we currently limit the size of an in-memory JAR to 1 MiB.
    private static final int MAX_JAR_BYTES = 1024 * 1024;

    /**
     * Reads the Dapp module from JAR bytes, in memory.
     * Note that a Dapp module is expected to specify a main class and contain at least one class.
     * 
     * @param jar The JAR bytes.
     * @return The module, or null if the contents of the JAR were insufficient for a Dapp.
     * @throws IOException An error occurred while reading the JAR contents.
     */
    public static ImmortalDappModule readFromJar(byte[] jar) throws IOException {
        LoadedJar loadedJar = LoadedJar.fromBytes(jar);
        Map<String, byte[]> classes = loadedJar.classBytesByQualifiedNames;
        String mainClass = loadedJar.mainClassName;
        // To be a valid Dapp, this must specify a main class and have at least one class.
        return ((null != mainClass) && !classes.isEmpty())
                ? new ImmortalDappModule(classes, mainClass)
                : null;
    }

    public static ImmortalDappModule fromImmortalClasses(Map<String, byte[]> classes, String mainClass)  {
        return new ImmortalDappModule(classes, mainClass);
    }


    public final Map<String, byte[]> classes;
    public final String mainClass;

    private ImmortalDappModule(Map<String, byte[]> classes, String mainClass) {
        this.classes = classes;
        this.mainClass = mainClass;
    }

    /**
     * Create the in-memory JAR containing all the classes in this module.
     */
    public byte[] createJar(long blockTimeStamp) throws IOException {
        // set jar file timestamp to block timestamp so the whole network is in agreement over this.
        FileTime timestamp = FileTime.fromMillis(blockTimeStamp);

        // manifest, we explicitly write it so that can can control its timestamps.
        Manifest manifest = new Manifest();
        manifest.getMainAttributes().put(Attributes.Name.MANIFEST_VERSION, "1.0");
        manifest.getMainAttributes().put(Attributes.Name.MAIN_CLASS, this.mainClass);

        ZipEntry manifestEntry = new ZipEntry(JarFile.MANIFEST_NAME);
        manifestEntry.setLastModifiedTime(timestamp);
        manifestEntry.setLastAccessTime(timestamp);
        manifestEntry.setCreationTime(timestamp);

        // Create a temporary memory location for this JAR.
        ByteArrayOutputStream tempJarStream = new ByteArrayOutputStream(MAX_JAR_BYTES);

        // create the jar file
        try (JarOutputStream target = new JarOutputStream(tempJarStream)) {
            // first, write the manifest file
            target.putNextEntry(manifestEntry);
            manifest.write(target);
            target.closeEntry();

            // add the classes
            for (String clazz : this.classes.keySet()) {
                JarEntry entry = new JarEntry(clazz.replace('.', '/') + ".class");
                entry.setLastModifiedTime(timestamp);
                entry.setLastAccessTime(timestamp);
                entry.setCreationTime(timestamp);
                target.putNextEntry(entry);
                target.write(this.classes.get(clazz));
                target.closeEntry();
            }
        }
        return tempJarStream.toByteArray();
    }
}
