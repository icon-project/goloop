package org.aion.avm.core.classloading;

import java.util.*;
import java.util.function.Function;

import org.aion.avm.core.arraywrapping.ArrayNameMapper;
import org.aion.avm.core.arraywrapping.ArrayWrappingClassGenerator;
import org.aion.avm.core.util.DebugNameResolver;
import i.PackageConstants;
import i.RuntimeAssertionError;


/**
 * NOTE:  This implementation assumes that the classes we are trying to load are "safe" in that they don't reference
 * anything we don't want this classloader to load.
 * While we originally imposed some of our isolation at the classloader level, we now assume that is done in the
 * bytecode instrumentation/analysis phase.
 */
public class AvmClassLoader extends ClassLoader {
    // The ENUM modifier is defined in Class, but that is private so here is our copy of the constant.
    private static final int CLASS_IS_ENUM = 0x00004000;

    // Bytecode Map of static class of Dapp
    private Map<String, byte[]> bytecodeMap;

    // List of dynamic class generation handlers
    private ArrayList<Function<String, byte[]>> handlers;

    // Class object cache
    private final Map<String, Class<?>> cache;

    /**
     * Constructs a new AVM class loader.
     *
     * @param parent The explicitly required parent for the contract-namespace code which is shared across all contracts.
     * @param bytecodeMap the transformed bytecode
     * @param handlers a list of handlers which can generate byte code for the given name.
     */
    public AvmClassLoader(AvmSharedClassLoader parent, Map<String, byte[]> bytecodeMap, ArrayList<Function<String, byte[]>> handlers) {
        super(parent);
        this.bytecodeMap = bytecodeMap;
        this.handlers = handlers;
        this.cache = new HashMap<>();

        registerHandlers();
    }

    /**
     * Constructs a new AVM class loader.
     *
     * @param parent The explicitly required parent for the contract-namespace code which is shared across all contracts.
     * @param bytecodeMap the transformed bytecode
     */
    public AvmClassLoader(AvmSharedClassLoader parent, Map<String, byte[]> bytecodeMap) {
        this(parent, bytecodeMap, new ArrayList<>());
    }

    private void registerHandlers(){
        // Array wrapper is the only handler of the dynamic class generation request.
        Function<String, byte[]> wrapperGenerator = (cName) -> ArrayWrappingClassGenerator.arrayWrappingFactory(cName, this);
        this.handlers.add(wrapperGenerator);
    }

    /**
     * Loads the class with the specified name.
     * This method will load three type of classes
     * a) User defined Dapp class
     * b) Per Dapp internal/api class
     * c) Dynamically generated user defined class
     *
     * Other class loading requests will be delegated to {@link AvmSharedClassLoader}
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
        // NOTE:  We override this, instead of findClass, since we want to circumvent the normal delegation process of class loaders.
        Class<?> result = null;
        boolean shouldResolve = resolve;

        // Contract classloader only load user Dapp classes and per Dapp internal/api classes
        // Non user classes will be delegated to shared class loader
        // We have a priority order to load:
        // 1) Cache
        // 2) Injected static code
        // 3) Dynamically generated
        if (this.cache.containsKey(name)) {
            result = this.cache.get(name);
            // We got this from the cache so don't resolve.
            shouldResolve = false;
        } else if (this.bytecodeMap.containsKey(name)) {
            byte[] injected = this.bytecodeMap.get(name);
            result = defineClass(name, injected, 0, injected.length);
            // Note that this class loader should only be able to see classes we have transformed.  This means no enums.
            RuntimeAssertionError.assertTrue(0 == (CLASS_IS_ENUM & result.getModifiers()));
            this.cache.put(name, result);
        } else if (isUserArrayWrapper(name)) {
            // Try dynamic generation
            for (Function<String, byte[]> handler : handlers) {
                byte[] code = handler.apply(name);
                if (code != null) {
                    result = defineClass(name, code, 0, code.length);
                    this.cache.put(name, result);
                    break;
                }
            }
        }else{
            // Delegate request to parent
            result = getParent().loadClass(name);
            // We got this from the parent so don't resolve.
            shouldResolve = false;
        }
        
        if ((null != result) && shouldResolve) {
            resolveClass(result);
        }

        if (null == result) {
            throw new ClassNotFoundException(name);
        }
        return result;
    }

    private boolean isUserArrayWrapper(String className) {
        if (className.startsWith(PackageConstants.kArrayWrapperUnifyingDotPrefix)) {
            return this.bytecodeMap.containsKey(ArrayNameMapper.getElementInterfaceName(className));
        } else if (className.startsWith(PackageConstants.kArrayWrapperDotPrefix + "$")) {
            return this.bytecodeMap.containsKey(ArrayNameMapper.getClassWrapperElementName(className));
        }
        // since it is not an array wrapper
        return false;
    }

    /**
     * A helper for tests which want to load a class by its pre-renamed name and also ensure that the receiver was the loader (didn't delegate).
     * 
     * @param originalClassName The pre-renamed class name (.-style).
     * @return The transformed/renamed class instance.
     * @throws ClassNotFoundException Underlying load failed.
     */
    public Class<?> loadUserClassByOriginalName(String originalClassName, boolean preserveDebuggability) throws ClassNotFoundException {
        String renamedClass = DebugNameResolver.getUserPackageDotPrefix(originalClassName, preserveDebuggability);
        Class<?> clazz = this.loadClass(renamedClass);
        RuntimeAssertionError.assertTrue(this == clazz.getClassLoader());
        return clazz;
    }

    //Internal
    public byte[] getUserClassBytecodeByOriginalName(String className, boolean preserveDebuggability) {
        return this.bytecodeMap.get(DebugNameResolver.getUserPackageDotPrefix(className, preserveDebuggability));
    }

    public byte[] getUserClassBytecode(String className){
        return this.bytecodeMap.get(className);
    }
}
