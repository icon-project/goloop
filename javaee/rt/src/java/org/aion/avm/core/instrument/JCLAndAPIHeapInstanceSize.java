package org.aion.avm.core.instrument;

import org.aion.avm.utilities.Utilities;

import java.util.Map;
import java.util.stream.Collectors;
import java.util.stream.Stream;

public class JCLAndAPIHeapInstanceSize {
    private static final int DEFAULT_OBJECT_ALLOCATION_SIZE = 16;
    // calculated based adding field sizes of Throwable to DEFAULT_OBJECT_ALLOCATION_SIZE
    private static final int DEFAULT_EXCEPTION_ALLOCATION_SIZE = 32;

    private static final Map<String, Integer> POST_RENAME_INSTANCE_SIZE = Stream.of(new Object[][]{
            {Utilities.fulllyQualifiedNameToInternalName(p.score.Address.class.getName()), 24}, // Object + byte[]
            {Utilities.fulllyQualifiedNameToInternalName(p.score.Result.class.getName()), 25}, //Object + boolean + byte[]
            {Utilities.fulllyQualifiedNameToInternalName(p.score.ValueBuffer.class.getName()), 24}, // Object + byte[]
            {Utilities.fulllyQualifiedNameToInternalName(s.java.lang.Class.class.getName()), 32}, // Object + Object + Object
            {Utilities.fulllyQualifiedNameToInternalName(s.java.lang.Enum.class.getName()), 28}, // Object + String + int
            {Utilities.fulllyQualifiedNameToInternalName(s.java.util.concurrent.TimeUnit.class.getName()), 28}, // Enum
            {Utilities.fulllyQualifiedNameToInternalName(s.java.math.RoundingMode.class.getName()), 32}, // Enum + int

            // non generated exception classes
            {Utilities.fulllyQualifiedNameToInternalName(s.java.lang.Throwable.class.getName()), DEFAULT_EXCEPTION_ALLOCATION_SIZE}, // Object + String + Object
            {Utilities.fulllyQualifiedNameToInternalName(s.java.lang.AssertionError.class.getName()), DEFAULT_EXCEPTION_ALLOCATION_SIZE}, // Throwable
            {Utilities.fulllyQualifiedNameToInternalName(s.java.lang.EnumConstantNotPresentException.class.getName()), 48}, // Throwable + Object + String
            {Utilities.fulllyQualifiedNameToInternalName(s.java.util.NoSuchElementException.class.getName()), DEFAULT_EXCEPTION_ALLOCATION_SIZE}, // Throwable
            {Utilities.fulllyQualifiedNameToInternalName(s.java.lang.TypeNotPresentException.class.getName()), 40}, // Throwable + String
            {Utilities.fulllyQualifiedNameToInternalName(s.java.lang.Error.class.getName()), DEFAULT_EXCEPTION_ALLOCATION_SIZE}, // Throwable
            {Utilities.fulllyQualifiedNameToInternalName(s.java.lang.Exception.class.getName()), DEFAULT_EXCEPTION_ALLOCATION_SIZE}, // Throwable
            {Utilities.fulllyQualifiedNameToInternalName(s.java.lang.RuntimeException.class.getName()), DEFAULT_EXCEPTION_ALLOCATION_SIZE}, // Throwable
            {Utilities.fulllyQualifiedNameToInternalName(s.score.RevertException.class.getName()), DEFAULT_EXCEPTION_ALLOCATION_SIZE}, // Throwable
            {Utilities.fulllyQualifiedNameToInternalName(s.score.ScoreRevertException.class.getName()), 36}, // Throwable + int
    }).collect(Collectors.toMap(data -> (String) data[0], data -> (Integer) data[1]));

    public static int getAllocationSizeForJCLAndAPISlashClass(String slashName) {
        return POST_RENAME_INSTANCE_SIZE.getOrDefault(slashName, DEFAULT_OBJECT_ALLOCATION_SIZE);
    }

    // returns the allocation size for our generated exceptions. This is set knowing that exceptions we produce do not have any non-static fields.
    public static int getAllocationSizeForGeneratedExceptionSlashClass() {
        return DEFAULT_EXCEPTION_ALLOCATION_SIZE;
    }
}
