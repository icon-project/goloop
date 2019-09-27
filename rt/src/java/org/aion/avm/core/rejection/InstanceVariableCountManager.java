package org.aion.avm.core.rejection;

import java.util.HashMap;
import java.util.Map;

import i.RuntimeAssertionError;

/**
 * Manages the logic around counting instance variables and imposing our limit.
 */
public class InstanceVariableCountManager {
    // We limit this to 31 since that avoids certain contrived cases where large objects can cause performance and billing problems.
    // (we only limit on the number of variables to keep the heuristic simple)
    private static final int MAX_INSTANCE_VARIABLES = 31;
    private final Map<String, Integer> nameToDeclaredCount = new HashMap<>();
    private final Map<String, String> nameToSuperClassName = new HashMap<>();

    public void addCount(String className, String superClassName, int count) {
        this.nameToDeclaredCount.put(className, count);
        this.nameToSuperClassName.put(className, superClassName);
    }

    public void verifyAllCounts() {
        Map<String, Integer> cache = new HashMap<>();
        for (String className : this.nameToDeclaredCount.keySet()) {
            int thisSize = populateAndCacheSize(cache, className);
            if (thisSize > MAX_INSTANCE_VARIABLES) {
                throw RejectedClassException.tooManyInstanceVariables(className);
            } else {
                // Verify that the other method cached this.
                RuntimeAssertionError.assertTrue(cache.containsKey(className));
            }
        }
    }


    private int populateAndCacheSize(Map<String, Integer> cache, String className) {
        int size = 0;
        // Terminate when we get outside of the user code (we will reference the type from the child but it won't be in the map).
        String superClassName = this.nameToSuperClassName.get(className);
        if (null != superClassName) {
            if (cache.containsKey(className)) {
                size = cache.get(className);
            } else {
                int parentSize = populateAndCacheSize(cache, superClassName);
                size = parentSize + this.nameToDeclaredCount.get(className);
                cache.put(className, size);
            }
        }
        return size;
    }
}
