package org.aion.avm.core.util;

import org.aion.types.AionAddress;
import p.avm.Blockchain;
import i.IBlockchainRuntime;
import i.IRuntimeSetup;
import org.aion.avm.core.ClassToolchain;
import org.aion.avm.core.classloading.AvmClassLoader;
import org.aion.avm.core.miscvisitors.ClassRenameVisitor;
import i.CommonInstrumentation;
import i.Helper;
import i.RuntimeAssertionError;
import i.StackWatcher;
import org.objectweb.asm.ClassReader;
import org.objectweb.asm.ClassWriter;

import java.io.*;
import java.lang.reflect.Field;
import java.lang.reflect.InvocationTargetException;
import java.security.SecureRandom;
import java.util.ArrayList;
import java.util.Collections;
import java.util.HashMap;
import java.util.List;
import java.util.Map;
import java.util.Set;


/**
 * Common utilities we often want to use in various tests (either temporarily or permanently).
 * These are kept here just to avoid duplication.
 */
public class Helpers {

    private static final char[] hexArray = "0123456789abcdef".toCharArray();
    public static final AionAddress ZERO_ADDRESS = address(0);

    /**
     * Converts byte array into its hex string representation.
     *
     * @param bytes
     * @return
     */
    public static String bytesToHexString(byte[] bytes) {
        if (bytes.length == 0){
            return "void";
        }

        int length = bytes.length;

        char[] hexChars = new char[length * 2];
        for (int i = 0; i < length; i++) {
            int v = bytes[i] & 0xFF;
            hexChars[i * 2] = hexArray[v >>> 4];
            hexChars[i * 2 + 1] = hexArray[v & 0x0F];
        }
        return new String(hexChars);
    }

    /**
     * Converts hex string into its byte[] representation.
     *
     * @param s
     * @return
     */
    public static byte[] hexStringToBytes(String s) {
        if (s.startsWith("0x")) {
            s = s.substring(2);
        }

        int len = s.length();
        byte[] data = new byte[len / 2];
        for (int i = 0; i < len; i += 2) {
            data[i / 2] = (byte) ((Character.digit(s.charAt(i), 16) << 4)
                    + Character.digit(s.charAt(i + 1), 16));
        }
        return data;
    }

    /**
     * Writes the given bytes to the file at the given path.
     * This is effective for dumping re-written bytecode to file for offline analysis.
     *
     * @param bytes The bytes to write.
     * @param path  The path where the file should be written.
     */
    public static void writeBytesToFile(byte[] bytes, String path) {
        File f = new File(path);
        f.getParentFile().mkdirs();
        try (FileOutputStream fos = new FileOutputStream(f)) {
            fos.write(bytes);
        } catch (IOException e) {
            // This is for tests so we aren't expecting the failure.
            throw RuntimeAssertionError.unexpected(e);
        }
    }

    /**
     * Reads file as a byte array.
     *
     * @param path
     * @return
     */
    public static byte[] readFileToBytes(String path) {
        File f = new File(path);
        byte[] b = new byte[(int) f.length()];

        try (DataInputStream in = new DataInputStream(new FileInputStream(f))) {
            in.readFully(b);
        } catch (IOException e) {
            throw RuntimeAssertionError.unexpected(e);
        }

        return b;
    }

    /**
     * A helper which will attempt to load the given resource path as bytes.
     * Returns null if the resource could not be found.
     *
     * @param resourcePath The path to this resource, within the parent class loader.
     * @return The resource as bytes, or null if not found.
     */
    public static byte[] loadRequiredResourceAsBytes(String resourcePath) {
        InputStream stream = Helpers.class.getClassLoader().getResourceAsStream(resourcePath);
        byte[] raw = null;
        if (null != stream) {
            try {
                raw = stream.readAllBytes();
            } catch (IOException e) {
                throw RuntimeAssertionError.unexpected(e);
            }
        }
        return raw;
    }

    private static SecureRandom secureRandom = new SecureRandom();

    public static AionAddress randomAddress() {
        byte[] bytes = new byte[AionAddress.LENGTH];
        secureRandom.nextBytes(bytes);
        return new AionAddress(bytes);
    }

    /**
     * Generate random byte array of the specified length.
     *
     * @param n
     * @return
     */
    public static byte[] randomBytes(int n) {
        byte[] bytes = new byte[n];
        secureRandom.nextBytes(bytes);

        return bytes;
    }

    /**
     * Converts a fully qualified class name into it's JVM internal form.
     *
     * @param fullyQualifiedName
     * @return
     */
    public static String fulllyQualifiedNameToInternalName(String fullyQualifiedName) {
        return fullyQualifiedName.replaceAll("\\.", "/");
    }

    /**
     * Converts a JVM internal class name into a fully qualified name.
     *
     * @param internalName
     * @return
     */
    public static String internalNameToFulllyQualifiedName(String internalName) {
        return internalName.replaceAll("/", ".");
    }

    private static String blockchainRuntimeClassName = Blockchain.class.getName();
    private static byte[] blockchainRuntimeBytes = Helpers.loadRequiredResourceAsBytes(blockchainRuntimeClassName.replaceAll("\\.", "/") + ".class");

    /**
     * A common helper used to construct a map of visible class bytecode for an AvmClassLoader instance.
     * Typically, this is used right before "instantiateHelper()", below (this creates/adds the class it loads).
     *
     * @param inputMap The initial map of class names to bytecodes.
     * @param helperBytes The bytecode of the Helper class (will be internally renamed to the appropriate name).
     * @return The inputMap with the Helper bytecode added.
     */
    public static Map<String, byte[]> mapIncludingHelperBytecode(Map<String, byte[]> inputMap, byte[] helperBytes) {
        // First, rename the helper class to the runtime helper name.
        byte[] renamedBytes = new ClassToolchain.Builder(helperBytes, ClassReader.SKIP_FRAMES | ClassReader.SKIP_DEBUG)
                        .addNextVisitor(new ClassRenameVisitor(Helper.RUNTIME_HELPER_NAME))
                        .addWriter(new ClassWriter(ClassWriter.COMPUTE_FRAMES | ClassWriter.COMPUTE_MAXS))
                        .build()
                        .runAndGetBytecode();
        
        // Now, construct the map.
        Map<String, byte[]> modifiedMap = new HashMap<>(inputMap);
        modifiedMap.put(Helper.RUNTIME_HELPER_NAME, renamedBytes);
        modifiedMap.put(blockchainRuntimeClassName, blockchainRuntimeBytes);
        return modifiedMap;
    }

    public static byte[] loadDefaultHelperBytecode() {
        String helperName = Helper.class.getName();
        String helperResourcePath = Helpers.fulllyQualifiedNameToInternalName(helperName) + ".class";
        return Helpers.loadRequiredResourceAsBytes(helperResourcePath);
    }

    /**
     * Attaches a Blockchain instance to the Blockchain class (per contract) so DApp can
     * access blockchain related methods.
     *
     * @param contractLoader
     * @param rt
     */
    public static void attachBlockchainRuntime(AvmClassLoader contractLoader, IBlockchainRuntime rt) {
        try {
            String runtimeClassName = Blockchain.class.getName();
            Class<?> helperClass = contractLoader.loadClass(runtimeClassName);
            helperClass.getField("blockchainRuntime").set(null, rt);
        } catch (Throwable t) {
            // Errors at this point imply something wrong with the installation so fail.
            throw RuntimeAssertionError.unexpected(t);
        }
    }

    // for testing purpose
    public static void attachStackWatcher(AvmClassLoader contractLoader, StackWatcher stackWatcher) {
        try {
            Class<?> helperClass = contractLoader.loadClass(Helper.RUNTIME_HELPER_NAME);
            Field targetField = helperClass.getDeclaredField("target");
            targetField.setAccessible(true);
            Field currentFrameField = contractLoader.loadClass(CommonInstrumentation.class.getName()).getDeclaredField("currentFrame");
            currentFrameField.setAccessible(true);
            Field stackWatcherField = contractLoader.loadClass(CommonInstrumentation.FrameState.class.getName()).getDeclaredField("stackWatcher");
            stackWatcherField.setAccessible(true);
            stackWatcherField.set(currentFrameField.get(targetField.get(null)), stackWatcher);
        } catch (Throwable t) {
            // Errors at this point imply something wrong with the installation so fail.
            throw RuntimeAssertionError.unexpected(t);
        }
    }

    // for test suites only
    public static AionAddress address(int n) {
        byte[] arr = new byte[32];
        for (int i = 0; i < arr.length; i++) {
            arr[i] = (byte) n;
        }
        return new AionAddress(arr);
    }

    public static byte[] merge(byte[]...arrays) {
        int length = 0;
        for (byte[] array : arrays) {
            length += array.length;
        }

        byte[] ret = new byte[length];
        int start = 0;
        for (byte[] array : arrays) {
            System.arraycopy(array, 0, ret, start, array.length);
            start += array.length;
        }

        return ret;
    }

    /**
     * Sorts the user contract class names given in "classNames", alphabetically, and then looks up each of their corresponding class objects in
     * classLoader.  Note that only class names within the "user" namspace are considered.
     * 
     * @param classLoader The class loader where the classes exist.
     * @param classNames The names of the classes which should be loaded.
     * @return The class objects, in alphabetical order by their names.
     */
    public static List<Class<?>> getAlphabeticalUserTransformedDappClasses(AvmClassLoader classLoader, Set<String> classNames) {
        List<String> nameList = new ArrayList<>(classNames);
        Collections.sort(nameList);
        List<Class<?>> classList = new ArrayList<>();
        for (String name : nameList) {
                try {
                    classList.add(classLoader.loadClass(name));
                } catch (ClassNotFoundException e) {
                    // We can't fail to find something which we know we put in there.
                    RuntimeAssertionError.unexpected(e);
                }
        }
        return classList;
    }

    /**
     * Instantiates the static instrumentation callout class ("H") within a given classloader, returning the new instance for attach/detach
     * within the static helper.
     * 
     * @param loader The class loader to search for the "H" class.
     * @return The instance which can be used to attach/detach the instrumentation helper class to an implementation.
     */
    public static IRuntimeSetup getSetupForLoader(ClassLoader loader) {
        try {
            String helperClassName = Helper.RUNTIME_HELPER_NAME;
            Class<?> clazz = loader.loadClass(helperClassName);
            RuntimeAssertionError.assertTrue(clazz.getClassLoader() == loader);
            return (IRuntimeSetup) clazz.getConstructor().newInstance();
        } catch (InstantiationException | IllegalAccessException | IllegalArgumentException | InvocationTargetException | NoSuchMethodException | SecurityException | ClassNotFoundException e) {
            // We require that this be instantiated in this way.
            throw RuntimeAssertionError.unexpected(e);
        }
    }
}
