package org.aion.avm.core.classloading;

import org.aion.avm.core.arraywrapping.ArrayWrappingClassGenerator;
import i.PackageConstants;
import i.RuntimeAssertionError;

import java.util.ArrayList;
import java.util.HashMap;
import java.util.Map;
import java.util.function.Function;


/**
 * This classloader is meant to sit as parent to AvmClassLoader and only exists to handle the common code which we generate and treat
 * as part of the contract namespace, but is common and class-immutable across all contracts.
 */
public class AvmSharedClassLoader extends ClassLoader {
    // Bytecode Map of shared avm static classes
    private final Map<String, byte[]> bytecodeMap;

    // Static class cache generated during initialization phase, after initialization it can provide lock free access
    private final Map<String, Class<?>> cacheStatic;

    // Dynamic class cache used for dynamic class generation, require lock before access
    private final Map<String, Class<?>> cacheDynamic;

    // List of dynamic class generation handlers
    private ArrayList<Function<String, byte[]>> handlers;

    // If the initialization of the NodeEnvironment is done
    private boolean initialized = false;

    /**
     * Constructs a new AVM shared class loader.
     *
     * @param bytecodeMap the shared class bytecodes
     */
    public AvmSharedClassLoader(Map<String, byte[]> bytecodeMap) {
        // Note that we will always descend from the classloader which loaded us (issue-331: can't assume this is the system loader).
        super(AvmClassLoader.class.getClassLoader());
        this.bytecodeMap = bytecodeMap;
        this.cacheStatic = new HashMap<>();
        this.cacheDynamic = new HashMap<>();
        this.handlers = new ArrayList<>();

        registerHandlers();
    }

    // Register runtime class generator
    private void registerHandlers(){
        Function<String, byte[]> wrapperGenerator = (cName) -> ArrayWrappingClassGenerator.arrayWrappingFactory(cName, this);
        this.handlers.add(wrapperGenerator);
    }

    /**
     * Inject classes into dynamic cache
     */
    public void putIntoDynamicCache(Class<?>[] classes){
        for (int i = 0; i < classes.length; i++){
            this.cacheDynamic.putIfAbsent(classes[i].getName(), classes[i]);
        }
    }

    /**
     * Inject classes into static cache
     */
    public void putIntoStaticCache(Class<?>[] classes){
        for (int i = 0; i < classes.length; i++){
            this.cacheStatic.putIfAbsent(classes[i].getName(), classes[i]);
        }
    }

    /**
     * Finish the initialization phase of the AVM shared class loader.
     * All classes in the code cache will be eagerly loaded.
     */
    public void finishInitialization() {
        for (String name: this.bytecodeMap.keySet()){
            try {
                this.loadClass(name, true);
            }catch (ClassNotFoundException e){
                RuntimeAssertionError.unreachable("Shared classloader initialization missing entry: " + name);
            }
        }
        this.initialized = true;
    }

    /**
     * Loads the class with the specified name.
     * This method will load two types of classes.
     * a) Statically generated shadow JCL classes
     * b) Dynamically generated shared classes (array wrappers)
     *
     * Other class loading requests will be delegated to its parent (By default, {@link ClassLoader})
     * Note that {@link AvmSharedClassLoader} will also cache the returned class object from its parent to speed up
     * concurrent class access.
     *
     * @param  name The name of the class
     *
     * @return  The resulting {@code Class} object
     *
     * @throws  ClassNotFoundException
     *          If the class was not found
     */
    @Override
    public Class<?> loadClass(String name, boolean resolve) throws ClassNotFoundException {
        Class<?> result = null;
        boolean shouldResolve = resolve;

        // All user space class should be loaded with Dapp loader
        if (name.contains(PackageConstants.kUserDotPrefix)){
            RuntimeAssertionError.unreachable("FAILED: Shared classloader receive request of: " + name);
        }

        // Array wrapper classes are either already in dynamic cache, or need to be generated
        if (name.startsWith(PackageConstants.kArrayWrapperDotPrefix) || name.startsWith(PackageConstants.kArrayWrapperUnifyingDotPrefix)){
            synchronized (this.cacheDynamic) {
                if (this.cacheDynamic.containsKey(name)) {
                    result = this.cacheDynamic.get(name);
                    // We got this from the cache so don't resolve.
                    shouldResolve = false;
                } else {
                    for (Function<String, byte[]> handler : handlers) {
                        byte[] code = handler.apply(name);
                        if (code != null) {
                            result = defineClass(name, code, 0, code.length);
                            this.cacheDynamic.putIfAbsent(name, result);
                            break;
                        }
                    }
                }
            }
        }else {
            // Initialization phase
            // Initialization is guaranteed to be single threaded
            if (!initialized) {
                if (this.cacheStatic.containsKey(name)) {
                    result = this.cacheStatic.get(name);
                    // We got this from the cache so don't resolve.
                    shouldResolve = false;
                } else if (this.bytecodeMap.containsKey(name)) {
                    byte[] injected = this.bytecodeMap.get(name);
                    result = defineClass(name, injected, 0, injected.length);
                    this.cacheStatic.putIfAbsent(name, result);
                } else {
                    // Cache miss, delegate it to parent
                    result = getParent().loadClass(name);
                    shouldResolve = false;
                    this.cacheStatic.putIfAbsent(name, result);
                }
            }

            // After initialization we have lock free static cache
            else {
                if (this.cacheStatic.containsKey(name)) {
                    result = this.cacheStatic.get(name);
                    shouldResolve = false;
                } else {
                    // Cache miss, delegate it to parent
                    result = getParent().loadClass(name);
                    shouldResolve = false;
                }
            }
        }

        if (null != result && shouldResolve) {
            resolveClass(result);
        }

        return result;
    }
}
