package org.aion.avm.core.verification;

import java.util.HashMap;
import java.util.Map;


/**
 * The restricted class loader we use when doing our pre-transform verification.
 * This case is one where we need lots of control and restriction since we are loading untrusted code:
 * IF THE &lt;CLINIT&gt; ROUTINE RUNS, THIS IS A SERIOUS BUG (GIVES USER CONTROL OF THE NODE).
 */
public class VerifierClassLoader extends ClassLoader {
    // Note that we explicitly only load each class once.
    private final Map<String, byte[]> notYetLoaded;
    private final Map<String, Class<?>> loaded;

    public VerifierClassLoader(Map<String, byte[]> classes) {
        // Note that we will always descend from the classloader which loaded us (issue-331: can't assume this is the system loader).
        super(VerifierClassLoader.class.getClassLoader());
        // We want to mutate this, so make a copy.
        this.notYetLoaded = new HashMap<>(classes);
        this.loaded = new HashMap<>();
    }

    @Override
    public Class<?> loadClass(String name, boolean resolve) throws ClassNotFoundException {
        // NOTE:  We override this, instead of findClass, since we want to circumvent the normal delegation process of class loaders.
        Class<?> result = null;
        boolean shouldResolve = resolve;
        
        if (this.loaded.containsKey(name)) {
            result = this.loaded.get(name);
            // We got this from the cache so don't resolve.
            shouldResolve = false;
        } else if (this.notYetLoaded.containsKey(name)) {
            // Remove this from the not yet loaded, load it, and add it to the loaded map.
            byte[] bytecode = this.notYetLoaded.remove(name);
            result = defineClass(name, bytecode, 0, bytecode.length);
            this.loaded.put(name, result);
        } else {
            // This might be in the parent.
            result = getParent().loadClass(name);
            // We got this from the parent so don't resolve.
            shouldResolve = false;
        }
        
        if ((null != result) && shouldResolve) {
            resolveClass(result);
        }
        if (null == result) {
            throw new ClassNotFoundException();
        }
        return result;
    }

    /**
     * Mostly used just to verify that we are done.
     * 
     * @return Out of the classes we were given, the number we haven't yet loaded.
     */
    public int getNotYetLoadedCount() {
        return this.notYetLoaded.size();
    }
}
