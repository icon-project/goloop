package org.aion.avm.core.persistence;

import java.io.IOException;
import java.lang.reflect.Field;
import java.lang.reflect.InvocationTargetException;
import java.lang.reflect.Method;
import java.lang.reflect.Modifier;
import java.nio.ByteBuffer;
import java.util.ArrayList;
import java.util.Arrays;
import java.util.HashMap;
import java.util.HashSet;
import java.util.List;
import java.util.Map;
import java.util.Set;

import foundation.icon.ee.types.ObjectGraph;
import foundation.icon.ee.types.DAppRuntimeState;
import foundation.icon.ee.util.MethodUnpacker;
import foundation.icon.ee.util.Unshadower;
import i.IInstrumentation;
import org.aion.avm.NameStyle;
import org.aion.avm.StorageFees;
import org.aion.avm.core.ClassRenamer;
import org.aion.avm.core.ClassRenamerBuilder;
import org.aion.avm.core.types.CommonType;
import org.aion.avm.core.util.DebugNameResolver;
import i.AvmThrowable;
import i.IBlockchainRuntime;
import i.IObjectDeserializer;
import i.IObjectSerializer;
import i.PackageConstants;
import p.avm.Blockchain;
import org.aion.avm.core.classloading.AvmClassLoader;
import org.aion.avm.core.util.Helpers;
import i.Helper;
import i.IRuntimeSetup;
import i.InternedClasses;
import i.MethodAccessException;
import i.RuntimeAssertionError;
import i.UncaughtException;


/**
 * Manages the organization of a DApp's root classes serialized shape as well as how to kick-off the serialization/deserialization
 * operations of the entire object graph (since both operations start at the root classes defined within the DApp).
 * Only the class statics and maybe a few specialized instances will be populated here.  The graph is limited by installing instance
 * stubs into fields pointing at objects.
 * 
 * We will store the data for all classes in a single storage key to avoid small IO operations when they are never used partially.
 * 
 * This class was originally just used to house the top-level calls related to serializing and deserializing a DApp but now it also
 * contains information relating to the DApp, in order to accomplish this.
 * Specifically, it now contains the ClassLoader, information about the class instances, and the cache of any reflection data.
 * NOTE:  It does NOT contain any information about the data currently stored within the Class objects associated with the DApp, nor
 * does it have any information about persisted aspects of the DApp (partly because it doesn't know anything about storage versioning).
 * 
 * NOTE:  Nothing here should be eagerly cached or looked up since the external caller is responsible for setting up the environment
 * such that it is fully usable.  Attempting to eagerly interact with it before then might not be safe.
 */
public class LoadedDApp {
    private static final String METHOD_PREFIX = "avm_";

    private static final Method SERIALIZE_SELF;
    private static final Method DESERIALIZE_SELF;
    private static final Field FIELD_READ_INDEX;
    
    static {
        try {
            Class<?> shadowObject = s.java.lang.Object.class;
            SERIALIZE_SELF = shadowObject.getDeclaredMethod("serializeSelf", Class.class, IObjectSerializer.class);
            DESERIALIZE_SELF = shadowObject.getDeclaredMethod("deserializeSelf", Class.class, IObjectDeserializer.class);
            FIELD_READ_INDEX = shadowObject.getDeclaredField("readIndex");
        } catch (NoSuchMethodException | SecurityException | NoSuchFieldException e) {
            // These are statically defined so can't fail.
            throw RuntimeAssertionError.unexpected(e);
        }
    }

    public final ClassLoader loader;
    // Note that the sortedUserClasses array does NOT include the constant class.
    private final Class<?>[] sortedUserClasses;
    private final Class<?> constantClass;
    private final String originalMainClassName;
    private final SortedFieldCache fieldCache;
    private final Map<String, foundation.icon.ee.types.Method> nameToMethod;

    // Other caches of specific pieces of data which are lazily built.
    public final IRuntimeSetup runtimeSetup;
    private Class<?> blockchainRuntimeClass;
    private Class<?> mainClass;
    private Field runtimeBlockchainRuntimeField;
    private long loadedDataBlockNum;
    private long loadedCodeBlockNum;

    // Note that we track the interned classes here since they have the same lifecycle as the LoadedDApp (including for reentrant calls).
    private InternedClasses internedClasses;

    private final ClassRenamer classRenamer;
    private final boolean preserveDebuggability;

    // Next hashcode which can be used to resume the state or serialize the DApp
    private int hashCode;
    // Used for billing
    private int serializedLength;

    /**
     * Creates the LoadedDApp to represent the classes related to DApp at address.
     * 
     * @param loader The class loader to look up shape.
     * @param userClasses The classes provided by the user.
     * @param constantClass The class we generated to contain all constants.
     * @param originalMainClassName The pre-translation name of the user's main class.
     * @param preserveDebuggability True if we should preserve debuggability by not renaming classes.
     */
    public LoadedDApp(ClassLoader loader, Class<?>[] userClasses, Class<?> constantClass, String originalMainClassName, byte[] apis, boolean preserveDebuggability) {
        this.loader = loader;
        // Note that the storage system defines the classes as being sorted alphabetically.
        this.sortedUserClasses = Arrays.stream(userClasses)
                .sorted((f1, f2) -> f1.getName().compareTo(f2.getName()))
                .toArray(Class[]::new);
        this.constantClass = constantClass;
        this.originalMainClassName = originalMainClassName;
        this.fieldCache = new SortedFieldCache(this.loader, SERIALIZE_SELF, DESERIALIZE_SELF, FIELD_READ_INDEX);
        this.preserveDebuggability = preserveDebuggability;

        // Collect all of the user-defined classes, discarding any generated exception wrappers for them.
        // This information is to be handed off to the persistance layer.
        Set<String> postRenameUserClasses = new HashSet<>();
        for (Class<?> userClass : this.sortedUserClasses) {
            String className = userClass.getName();
            if (!className.startsWith(PackageConstants.kExceptionWrapperDotPrefix)) {
                postRenameUserClasses.add(className);
            }
        }

        this.classRenamer = new ClassRenamerBuilder(NameStyle.DOT_NAME, this.preserveDebuggability)
            .loadPostRenameUserDefinedClasses(postRenameUserClasses)
            .loadPreRenameJclExceptionClasses(fetchPreRenameSlashStyleJclExceptions())
            .prohibitExceptionWrappers()
            .prohibitUnifyingArrayTypes()
            .build();
        
        // We also know that we need the runtimeSetup, meaning we also need the helperClass.
        try {
            String helperClassName = Helper.RUNTIME_HELPER_NAME;
            Class<?> helperClass = this.loader.loadClass(helperClassName);
            RuntimeAssertionError.assertTrue(helperClass.getClassLoader() == this.loader);
            this.runtimeSetup = (IRuntimeSetup) helperClass.getConstructor().newInstance();
        } catch (InstantiationException | IllegalAccessException | IllegalArgumentException | InvocationTargetException | NoSuchMethodException | SecurityException | ClassNotFoundException e) {
            // We require that this be instantiated in this way.
            throw RuntimeAssertionError.unexpected(e);
        }
        loadedDataBlockNum = -1;
        loadedCodeBlockNum = -1;
        this.internedClasses = new InternedClasses();
        nameToMethod = new HashMap<>();
        try {
            var methods = MethodUnpacker.readFrom(apis);
            for (var m : methods) {
                nameToMethod.put(m.getName(), m);
            }
        } catch (IOException e) {
            throw RuntimeAssertionError.unexpected(e);
        }
    }

    public Method getExternalMethod(foundation.icon.ee.types.Method m) throws ReflectiveOperationException  {
        var paramClasses = m.getParameterClasses();
        Class<?> clazz = loadMainClass();
        return clazz.getMethod(METHOD_PREFIX + m.getName(), paramClasses);
    }

    public void verifyExternalMethods() throws ReflectiveOperationException {
        Class<?> clazz = loadMainClass();
        for (var m : nameToMethod.entrySet()) {
            if (m.getValue().getType() != foundation.icon.ee.types.Method.MethodType.EVENT) {
                Method method = getExternalMethod(m.getValue());
                if (!Modifier.isStatic(method.getModifiers())) {
                    throw new NoSuchMethodException(String.format("method %s is not static", m.getKey()));
                }
            }
        }
    }

    /**
     * Requests that the Classes in the receiver be populated with data from the rawGraphData.
     * NOTE:  The caller is expected to manage billing - none of that is done in here.
     * 
     * @param internedClassMap The interned classes, in case class references need to be instantiated.
     * @param rawGraphData The data from which to read the graph (note that this must encompass all and only a completely serialized graph.
     * @return The nextHashCode serialized within the graph.
     */
    public int loadEntireGraph(InternedClasses internedClassMap, byte[] rawGraphData) {
        ByteBuffer inputBuffer = ByteBuffer.wrap(rawGraphData);
        List<Object> existingObjectIndex = null;
        StandardGlobalResolver resolver = new StandardGlobalResolver(internedClassMap, this.loader);
        StandardNameMapper classNameMapper = new StandardNameMapper(this.classRenamer);
        return Deserializer.deserializeEntireGraphAndNextHashCode(inputBuffer, existingObjectIndex, resolver,
                this.fieldCache, classNameMapper, this.sortedUserClasses, this.constantClass);
    }

    public int loadRuntimeState(DAppRuntimeState state) {
        ByteBuffer inputBuffer = ByteBuffer.wrap(state.getGraph().getRawData());
        List<Object> existingObjectIndex = state.getObjects();
        StandardGlobalResolver resolver = new StandardGlobalResolver(internedClasses, this.loader);
        StandardNameMapper classNameMapper = new StandardNameMapper(this.classRenamer);
        return Deserializer.deserializeEntireGraphAndNextHashCode(inputBuffer, existingObjectIndex, resolver,
                this.fieldCache, classNameMapper, this.sortedUserClasses, this.constantClass);
    }

    public Object deserializeObject(InternedClasses internedClassMap, byte[] rawGraphData) {
        ByteBuffer inputBuffer = ByteBuffer.wrap(rawGraphData);
        StandardGlobalResolver resolver = new StandardGlobalResolver(internedClassMap, this.loader);
        StandardNameMapper classNameMapper = new StandardNameMapper(this.classRenamer);
        return Deserializer.deserializeObject(inputBuffer, resolver, this.fieldCache, classNameMapper);
    }

    /**
     * Requests that the Classes in the receiver be walked and all referenced objects be serialized into a graph.
     * NOTE:  The caller is expected to manage billing - none of that is done in here.
     * 
     * @param nextHashCode The nextHashCode to serialize into the graph so that this can be resumed in the future.
     * @param maximumSizeInBytes The size limit on the serialized graph size (this is a parameter for testing but also to allow the caller to impose energy-based limits).
     * @return The enter serialized object graph.
     */
    public byte[] saveEntireGraph(int nextHashCode, int maximumSizeInBytes) {
        ByteBuffer outputBuffer = ByteBuffer.allocate(maximumSizeInBytes);
        List<Object> out_instanceIndex = null;
        List<Integer> out_calleeToCallerIndexMap = null;
        StandardGlobalResolver resolver = new StandardGlobalResolver(null, this.loader);
        StandardNameMapper classNameMapper = new StandardNameMapper(this.classRenamer);
        Serializer.serializeEntireGraph(outputBuffer, out_instanceIndex, out_calleeToCallerIndexMap, resolver, this.fieldCache, classNameMapper, nextHashCode, this.sortedUserClasses, this.constantClass);
        
        byte[] finalBytes = new byte[outputBuffer.position()];
        System.arraycopy(outputBuffer.array(), 0, finalBytes, 0, finalBytes.length);
        return finalBytes;
    }

    public DAppRuntimeState saveRuntimeState() {
        var hash = IInstrumentation.attachedThreadInstrumentation.get().peekNextHashCode();
        return saveRuntimeState(hash, StorageFees.MAX_GRAPH_SIZE);
    }

    public DAppRuntimeState saveRuntimeState(int nextHashCode, int maximumSizeInBytes) {
        ByteBuffer outputBuffer = ByteBuffer.allocate(maximumSizeInBytes);
        List<Object> out_instanceIndex = new ArrayList<>();
        List<Integer> out_calleeToCallerIndexMap = null;
        StandardGlobalResolver resolver = new StandardGlobalResolver(null, this.loader);
        StandardNameMapper classNameMapper = new StandardNameMapper(this.classRenamer);
        Serializer.serializeEntireGraph(outputBuffer, out_instanceIndex, out_calleeToCallerIndexMap, resolver, this.fieldCache, classNameMapper, nextHashCode, this.sortedUserClasses, this.constantClass);

        byte[] finalBytes = new byte[outputBuffer.position()];
        System.arraycopy(outputBuffer.array(), 0, finalBytes, 0, finalBytes.length);
        return new DAppRuntimeState(out_instanceIndex, ObjectGraph.getInstance(finalBytes));
    }

    public byte[] serializeObject(Object v) {
        return serializeObject(v, StorageFees.MAX_GRAPH_SIZE);
    }

    public byte[] serializeObject(Object v, int maximumSizeInBytes) {
        ByteBuffer outputBuffer = ByteBuffer.allocate(maximumSizeInBytes);
        StandardGlobalResolver resolver = new StandardGlobalResolver(null, this.loader);
        StandardNameMapper classNameMapper = new StandardNameMapper(this.classRenamer);
        Serializer.serializeObject(outputBuffer, v, resolver, this.fieldCache, classNameMapper);
        byte[] finalBytes = new byte[outputBuffer.position()];
        System.arraycopy(outputBuffer.array(), 0, finalBytes, 0, finalBytes.length);
        return finalBytes;
    }

    public ReentrantGraph captureStateAsCaller(int nextHashCode, int maxGraphSize) {
        StandardGlobalResolver resolver = new StandardGlobalResolver(null, this.loader);
        StandardNameMapper classNameMapper = new StandardNameMapper(this.classRenamer);
        return ReentrantGraph.captureCallerState(resolver, this.fieldCache, classNameMapper, maxGraphSize, nextHashCode, this.sortedUserClasses, this.constantClass);
    }

    public ReentrantGraph captureStateAsCallee(int updatedNextHashCode, int maxGraphSize) {
        StandardGlobalResolver resolver = new StandardGlobalResolver(null, this.loader);
        StandardNameMapper classNameMapper = new StandardNameMapper(this.classRenamer);
        return ReentrantGraph.captureCalleeState(resolver, this.fieldCache, classNameMapper, maxGraphSize, updatedNextHashCode, this.sortedUserClasses, this.constantClass);
    }

    public void commitReentrantChanges(InternedClasses internedClassMap, ReentrantGraph callerState, ReentrantGraph calleeState) {
        StandardGlobalResolver resolver = new StandardGlobalResolver(internedClassMap, this.loader);
        StandardNameMapper classNameMapper = new StandardNameMapper(this.classRenamer);
        callerState.commitChangesToState(resolver, this.fieldCache, classNameMapper, this.sortedUserClasses, this.constantClass, calleeState);
    }

    public void revertToCallerState(InternedClasses internedClassMap, ReentrantGraph callerState) {
        StandardGlobalResolver resolver = new StandardGlobalResolver(internedClassMap, this.loader);
        StandardNameMapper classNameMapper = new StandardNameMapper(this.classRenamer);
        callerState.revertChangesToState(resolver, this.fieldCache, classNameMapper, this.sortedUserClasses, this.constantClass);
    }

    /**
     * Attaches an IBlockchainRuntime instance to the Helper class (per contract) so DApp can
     * access blockchain related methods.
     *
     * Returns the previously attached IBlockchainRuntime instance if one existed, or null otherwise.
     *
     * NOTE:  The current implementation is mostly cloned from Helpers.attachBlockchainRuntime() but we will inline/cache more of this,
     * over time, and that older implementation is only used by tests (which may be ported to use this).
     *
     * @param runtime The runtime to install in the DApp.
     * @return The previously attached IBlockchainRuntime instance or null if none.
     */
    public IBlockchainRuntime attachBlockchainRuntime(IBlockchainRuntime runtime) {
        try {
            Field field = getBlochchainRuntimeField();
            IBlockchainRuntime previousBlockchainRuntime = (IBlockchainRuntime) field.get(null);
            field.set(null, runtime);
            return previousBlockchainRuntime;
        } catch (Throwable t) {
            // Errors at this point imply something wrong with the installation so fail.
            throw RuntimeAssertionError.unexpected(t);
        }
    }

    public Object callMethod(String methodName, Object[] params) throws Throwable {
        try {
            var m = nameToMethod.get(methodName);
            if (m==null) {
                throw new NoSuchMethodException(String.format("method %s is not in APIS", methodName));
            }
            Method method = getExternalMethod(m);
            Object sres = method.invoke(null, m.convertParameters(params));
            Object res;
            if (m.hasValidPrimitiveReturnType()) {
                res = sres;
            } else {
                try {
                    res = Unshadower.unshadow((s.java.lang.Object)sres);
                } catch (IllegalArgumentException e) {
                    throw new ClassCastException("bad return value");
                }
            }
            return res;
        } catch (ClassNotFoundException | SecurityException | ExceptionInInitializerError e) {
            // should have been handled during CREATE.
            RuntimeAssertionError.unexpected(e);

        } catch (NoSuchMethodException | IllegalAccessException e) {
            throw new MethodAccessException(e);

        } catch (InvocationTargetException e) {
            // handle the real exception
            if (e.getTargetException() instanceof UncaughtException) {
                handleUncaughtException(e.getTargetException().getCause());
            } else {
                handleUncaughtException(e.getTargetException());
            }
        }

        return null;
    }

    /**
     * Forces all the classes defined within this DApp to be loaded and initialized (meaning each has its &lt;clinit&gt; called).
     * This is called during the create action to force the DApp initialization code to be run before it is stripped off for
     * long-term storage.
     */
    public void forceInitializeAllClasses() throws Throwable {
        forceInitializeOneClass(this.constantClass);
        for (Class<?> clazz : this.sortedUserClasses) {
            forceInitializeOneClass(clazz);
        }
    }

    private void forceInitializeOneClass(Class<?> clazz) throws Throwable {
        try {
            Class<?> initialized = Class.forName(clazz.getName(), true, this.loader);
            // These must be the same instances we started with and they must have been loaded by this loader.
            RuntimeAssertionError.assertTrue(clazz == initialized);
            RuntimeAssertionError.assertTrue(initialized.getClassLoader() == this.loader);
        } catch (ClassNotFoundException e) {
            // This error would mean that this is assembled completely incorrectly, which is a static error in our implementation.
            RuntimeAssertionError.unexpected(e);
        } catch (SecurityException e) {
            // This would mean that the shadowing is not working properly.
            RuntimeAssertionError.unexpected(e);
        } catch (ExceptionInInitializerError e) {
            // handle the real exception
            handleUncaughtException(e.getException());
        } catch (Throwable t) {
            // Some other exceptions can float out from the user clinit, not always wrapped in ExceptionInInitializerError.
            handleUncaughtException(t);
        }
    }

    /**
     * The exception could be any {@link i.AvmThrowable}, any {@link java.lang.RuntimeException},
     * or a {@link e.s.java.lang.Throwable}.
     */
    private void handleUncaughtException(Throwable cause) throws Throwable {
        // thrown by us
        if (cause instanceof AvmThrowable) {
            throw cause;

            // thrown by runtime, but is never handled
        } else if ((cause instanceof RuntimeException) || (cause instanceof Error)) {
            throw new UncaughtException(cause);

            // thrown by users
        } else if (cause instanceof e.s.java.lang.Throwable) {
            // Note that we will need to unwrap this since the wrapper doesn't actually communicate anything, just being
            // used to satisfy Java exception relationship requirements (the user code populates the wrapped object).
            throw new UncaughtException(((e.s.java.lang.Throwable) cause).unwrap().toString(), cause);

        } else {
            RuntimeAssertionError.unexpected(cause);
        }
    }

    /**
     * Called before the DApp is about to be put into a cache.  This is so it can put itself into a "resumable" state.
     */
    public void clearDataState() {
        loadedDataBlockNum = -1;
        Deserializer.cleanClassStatics(this.fieldCache, this.sortedUserClasses, this.constantClass);
    }


    private Class<?> loadBlockchainRuntimeClass() throws ClassNotFoundException {
        Class<?> runtimeClass = this.blockchainRuntimeClass;
        if (null == runtimeClass) {
            String runtimeClassName = Blockchain.class.getName();
            runtimeClass = this.loader.loadClass(runtimeClassName);
            RuntimeAssertionError.assertTrue(runtimeClass.getClassLoader() == this.loader);
            this.blockchainRuntimeClass = runtimeClass;
        }
        return runtimeClass;
    }

    private Class<?> loadMainClass() throws ClassNotFoundException {
        Class<?> mainClass = this.mainClass;
        if (null == mainClass) {
            String mappedUserMainClass = DebugNameResolver.getUserPackageDotPrefix(this.originalMainClassName, this.preserveDebuggability);
            mainClass = this.loader.loadClass(mappedUserMainClass);
            RuntimeAssertionError.assertTrue(mainClass.getClassLoader() == this.loader);
            this.mainClass = mainClass;
        }
        return mainClass;
    }

    private Field getBlochchainRuntimeField() throws ClassNotFoundException, NoSuchFieldException, SecurityException  {
        Field runtimeBlockchainRuntimeField = this.runtimeBlockchainRuntimeField;
        if (null == runtimeBlockchainRuntimeField) {
            Class<?> runtimeClass = loadBlockchainRuntimeClass();
            runtimeBlockchainRuntimeField = runtimeClass.getField("blockchainRuntime");
            this.runtimeBlockchainRuntimeField = runtimeBlockchainRuntimeField;
        }
        return runtimeBlockchainRuntimeField;
    }

    /**
     * Dump the transformed class files of the loaded Dapp.
     * The output class files will be put under {@param path}.
     *
     * @param path The runtime to install in the DApp.
     */
    public void dumpTransformedByteCode(String path){
        AvmClassLoader appLoader = (AvmClassLoader) loader;
        dumpOneTransformedClass(path, appLoader, this.constantClass);
        for (Class<?> clazz : this.sortedUserClasses){
            dumpOneTransformedClass(path, appLoader, clazz);
        }
    }

    private void dumpOneTransformedClass(String path, AvmClassLoader appLoader, Class<?> clazz) {
        byte[] bytecode = appLoader.getUserClassBytecode(clazz.getName());
        String output = path + "/" + clazz.getName() + ".class";
        Helpers.writeBytesToFile(bytecode, output);
    }

    public void setLoadedCodeBlockNum(long loadedBlockNum) {
        loadedCodeBlockNum = loadedBlockNum;
    }

    public long getLoadedCodeBlockNum() {
        return loadedCodeBlockNum;
    }

    public void updateLoadedBlockForSuccessfulTransaction(long loadedBlockNum){
        // Store the current block as the last number which the DApp data was loaded in
        loadedDataBlockNum = loadedBlockNum;
    }

    public boolean hasValidCachedData(long loadedBlockNum){
        // Ensure data has been updated before the current block and it has not been reset after.
        // Note that from the time the data cache is updated, loadedDataBlockNum >= loadedCodeBlockNum
        return loadedDataBlockNum < loadedBlockNum && loadedDataBlockNum != -1;
    }

    public boolean hasValidCachedCode(long loadedBlockNum){
        // Ensure data has been updated before the current block and it has not been reset after.
        return loadedCodeBlockNum < loadedBlockNum && loadedCodeBlockNum != -1;
    }

    public void setHashCode(int hashCode) { this.hashCode = hashCode; }

    public void setSerializedLength(int serializedLength) { this.serializedLength = serializedLength; }

    public int getHashCode() { return hashCode; }

    public int getSerializedLength() { return serializedLength; }

    private Set<String> fetchPreRenameSlashStyleJclExceptions() {
        Set<String> jclExceptions = new HashSet<>();

        for (CommonType type : CommonType.values()) {
            if (type.isShadowException) {
                jclExceptions.add(type.dotName.substring(PackageConstants.kShadowDotPrefix.length()).replaceAll("\\.", "/"));
            }
        }

        return jclExceptions;
    }

    public InternedClasses getInternedClasses() {
        return internedClasses;
    }
}
