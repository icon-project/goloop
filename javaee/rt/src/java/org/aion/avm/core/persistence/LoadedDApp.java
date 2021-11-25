package org.aion.avm.core.persistence;

import foundation.icon.ee.score.ScoreClass;
import foundation.icon.ee.types.DAppRuntimeState;
import foundation.icon.ee.types.IllegalFormatException;
import foundation.icon.ee.types.ManualRevertException;
import foundation.icon.ee.types.ObjectGraph;
import foundation.icon.ee.types.Status;
import foundation.icon.ee.types.UnknownFailureException;
import foundation.icon.ee.util.MethodUnpacker;
import foundation.icon.ee.util.Shadower;
import foundation.icon.ee.util.Unshadower;
import i.AvmThrowable;
import i.GenericPredefinedException;
import i.Helper;
import i.IBlockchainRuntime;
import i.IInstrumentation;
import i.IObjectDeserializer;
import i.IObjectSerializer;
import i.IRuntimeSetup;
import i.InternedClasses;
import i.PackageConstants;
import i.RuntimeAssertionError;
import i.UncaughtException;
import org.aion.avm.NameStyle;
import org.aion.avm.StorageFees;
import org.aion.avm.core.ClassRenamer;
import org.aion.avm.core.ClassRenamerBuilder;
import org.aion.avm.core.IExternalState;
import org.aion.avm.core.types.CommonType;
import org.aion.avm.core.util.DebugNameResolver;
import p.score.Context;

import java.io.IOException;
import java.lang.reflect.Constructor;
import java.lang.reflect.Field;
import java.lang.reflect.InvocationTargetException;
import java.lang.reflect.Method;
import java.lang.reflect.Modifier;
import java.nio.ByteBuffer;
import java.util.ArrayList;
import java.util.Arrays;
import java.util.Comparator;
import java.util.HashMap;
import java.util.HashSet;
import java.util.List;
import java.util.Map;
import java.util.Set;

import static foundation.icon.ee.types.Method.MethodType;

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
    private final Map<String, Method> nameToMethod;
    private Constructor<?> ctor = null;

    // Other caches of specific pieces of data which are lazily built.
    public final IRuntimeSetup runtimeSetup;
    private Class<?> blockchainRuntimeClass;
    private Class<?> mainClass;
    private Field runtimeBlockchainRuntimeField;

    // Note that we track the interned classes here since they have the same lifecycle as the LoadedDApp (including for reentrant calls).
    private final InternedClasses internedClasses;

    private final ClassRenamer classRenamer;
    private final boolean preserveDebuggability;

    private Object mainInstance;
    private DAppRuntimeState stateCache;

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
                .sorted(Comparator.comparing(Class::getName))
                .toArray(Class[]::new);
        this.constantClass = constantClass;
        this.originalMainClassName = originalMainClassName;
        this.fieldCache = new SortedFieldCache(this.loader, SERIALIZE_SELF, DESERIALIZE_SELF, FIELD_READ_INDEX);
        this.preserveDebuggability = preserveDebuggability;

        // Collect all of the user-defined classes, discarding any generated exception wrappers for them.
        // This information is to be handed off to the persistence layer.
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
        this.internedClasses = new InternedClasses();
        nameToMethod = new HashMap<>();
        try {
            var methods = MethodUnpacker.readFrom(apis);
            var sc = new ScoreClass(loadMainClass());
            for (var m : methods) {
                if (m.getType() == MethodType.EVENT) {
                    continue;
                }
                if (m.getName().equals("<init>")) {
                    ctor = sc.findConstructor(m);
                } else {
                    var found = sc.findMethod(m);
                    if (found == null
                            || (found.getModifiers() & Modifier.STATIC) != 0
                            || (found.getModifiers() & Modifier.PUBLIC) == 0) {
                        throw new IllegalFormatException("Bad external method");
                    }
                    nameToMethod.put(m.getName(), found);
                }
            }
            if (ctor == null) {
                throw new IllegalFormatException("Bad constructor");
            }
        } catch (IOException e) {
            throw RuntimeAssertionError.unexpected(e);
        } catch (ClassNotFoundException e) {
            e.printStackTrace();
        }
    }

    private static IllegalFormatException fail(String fmt, Object... args) {
        throw new IllegalFormatException(String.format(fmt, args));
    }

    public DAppRuntimeState loadRuntimeState(IExternalState es) {
        if (stateCache != null) {
            var gh = stateCache.getGraph().getGraphHash();
            if (Arrays.equals(gh, es.getObjectGraphHash())) {
                stateCache = new DAppRuntimeState(stateCache, es.getNextHash());
                return stateCache;
            }
        }
        var graph = es.getObjectGraph();
        var rs = new DAppRuntimeState(null, graph);
        loadRuntimeState(rs);
        // to make object list
        return saveRuntimeState(rs.getGraph().getNextHash(),
                StorageFees.MAX_GRAPH_SIZE);
    }

    public void loadRuntimeState(DAppRuntimeState state) {
        if (stateCache == state) {
            return;
        }
        ByteBuffer inputBuffer = ByteBuffer.wrap(state.getGraph().getGraphData());
        List<Object> existingObjectIndex = state.getObjects();
        StandardGlobalResolver resolver = new StandardGlobalResolver(internedClasses, this.loader);
        StandardNameMapper classNameMapper = new StandardNameMapper(this.classRenamer);
        var buf = new Object[1];
        Deserializer.deserializeEntireGraph(inputBuffer, existingObjectIndex,
                resolver, this.fieldCache, classNameMapper,
                this.sortedUserClasses, this.constantClass, buf);
        mainInstance = buf[0];
        stateCache = state;
    }

    public DAppRuntimeState saveRuntimeState() {
        var hash = IInstrumentation.attachedThreadInstrumentation.get().peekNextHashCode();
        return saveRuntimeState(hash, StorageFees.MAX_GRAPH_SIZE);
    }

    public DAppRuntimeState saveRuntimeState(int nextHashCode, int maximumSizeInBytes) {
        ByteBuffer outputBuffer = ByteBuffer.allocate(maximumSizeInBytes);
        List<Object> out_instanceIndex = new ArrayList<>();
        StandardGlobalResolver resolver = new StandardGlobalResolver(null, this.loader);
        StandardNameMapper classNameMapper = new StandardNameMapper(this.classRenamer);
        Serializer.serializeEntireGraph(outputBuffer, out_instanceIndex,
                null, resolver, this.fieldCache,
                classNameMapper, this.sortedUserClasses, this.constantClass,
                this.mainInstance);
        byte[] finalBytes = new byte[outputBuffer.position()];
        System.arraycopy(outputBuffer.array(), 0, finalBytes, 0, finalBytes.length);
        stateCache = new DAppRuntimeState(out_instanceIndex,
                new ObjectGraph(nextHashCode, finalBytes));
        return stateCache;
    }

    public void invalidateStateCache() {
        stateCache = null;
    }

    public boolean hasSameGraphHash(byte[] graphHash) {
        if (stateCache == null) {
            return false;
        }
        var gh = stateCache.getGraph().getGraphHash();
        return Arrays.equals(gh, graphHash);
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

    public void initMainInstance(Object []params) throws AvmThrowable {
        try {
            mainInstance = ctor.newInstance(
                    Shadower.shadowObjects(params, ctor.getParameterTypes()));
        } catch (ArithmeticException e) {
            throw new GenericPredefinedException(Status.InvalidParameter, e);
        } catch (InvocationTargetException e) {
            handleUncaughtException(e.getTargetException());
        } catch (ExceptionInInitializerError e) {
            handleUncaughtException(e.getException());
        } catch (ReflectiveOperationException e) {
            throw new IllegalFormatException("cannot call constructor", e);
        } catch (IllegalArgumentException e) {
            RuntimeAssertionError.unexpected(e);
        }
    }

    public Object callMethod(String methodName, Object[] params) throws AvmThrowable {
        stateCache = null;
        try {
            Method method = nameToMethod.get(methodName);
            Object sres = method.invoke(mainInstance,
                    Shadower.shadowObjects(params, method.getParameterTypes()));
            try {
                return Unshadower.unshadow(sres);
            } catch (IllegalArgumentException e) {
                throw new UnknownFailureException("invalid return value");
            }
        } catch (ArithmeticException e) {
            throw new GenericPredefinedException(Status.InvalidParameter, e);
        } catch (InvocationTargetException e) {
            handleUncaughtException(e.getTargetException());
        } catch (ExceptionInInitializerError e) {
            handleUncaughtException(e.getException());
        } catch (ReflectiveOperationException | IllegalArgumentException e) {
            throw RuntimeAssertionError.unexpected(e);
        }
        return null;
    }

    /**
     * Forces all the classes defined within this DApp to be loaded and initialized (meaning each has its &lt;clinit&gt; called).
     * This is called during the create action to force the DApp initialization code to be run before it is stripped off for
     * long-term storage.
     */
    public void forceInitializeAllClasses() throws AvmThrowable {
        forceInitializeOneClass(this.constantClass);
        for (Class<?> clazz : this.sortedUserClasses) {
            forceInitializeOneClass(clazz);
        }
    }

    private void forceInitializeOneClass(Class<?> clazz) throws AvmThrowable {
        try {
            Class<?> initialized = Class.forName(clazz.getName(), true, this.loader);
            RuntimeAssertionError.assertTrue(clazz == initialized);
            RuntimeAssertionError.assertTrue(initialized.getClassLoader() == this.loader);
        } catch (ExceptionInInitializerError e) {
            handleUncaughtException(e.getException());
        } catch (ClassNotFoundException | LinkageError e) {
            throw new UnknownFailureException(e);
        } catch (SecurityException e) {
            throw RuntimeAssertionError.unexpected(e);
        }
    }

    /**
     * The exception could be any {@link i.AvmThrowable}, any {@link java.lang.RuntimeException},
     * or a {@link e.s.java.lang.Throwable}.
     */
    private void handleUncaughtException(Throwable cause) throws AvmThrowable {
        // thrown by us
        if (cause instanceof AvmThrowable) {
            throw (AvmThrowable)cause;
        }
        // thrown by runtime, but is never handled
        else if ((cause instanceof RuntimeException) || (cause instanceof Error)) {
            throw new UncaughtException(cause);
        }
        // thrown by users
        else if (cause instanceof e.s.java.lang.Throwable) {
            // Note that we will need to unwrap this since the wrapper doesn't actually communicate anything, just being
            // used to satisfy Java exception relationship requirements (the user code populates the wrapped object).
            var esThrowable = ((e.s.java.lang.Throwable) cause);
            var shadow = esThrowable.unwrap();
            if (shadow instanceof s.score.UserRevertException) {
                var e = (s.score.UserRevertException) shadow;
                throw new ManualRevertException(Status.fromUserCode(e.avm_getCode()), e.getMessage(), cause);
            }
            throw new UncaughtException(((e.s.java.lang.Throwable) cause).unwrap().toString(), cause);
        } else {
            RuntimeAssertionError.unexpected(cause);
        }
    }

    private Class<?> loadBlockchainRuntimeClass() throws ClassNotFoundException {
        Class<?> runtimeClass = this.blockchainRuntimeClass;
        if (null == runtimeClass) {
            String runtimeClassName = Context.class.getName();
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
