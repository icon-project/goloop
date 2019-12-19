package org.aion.avm.core.persistence;


/**
 * Used by the persistence system when reading or writing object instances in order to handle constants and classes.
 * These are both concepts which must be translated between the global in-memory state of the system and the serialized
 * for since they live longer than the invocation, itself.
 */
public interface IGlobalResolver {
    String getAsInternalClassName(Object target);
    Object getClassObjectForInternalName(String internalClassName);

    int getAsConstant(Object target);
    Object getConstantForIdentifier(int constantIdentifier);
}
