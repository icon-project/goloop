package org.aion.avm.core.persistence;


/**
 * Used by the persistence system to translate class names between the ones in the storage system and our internal names
 * which are specific to our implementation.
 */
public interface IPersistenceNameMapper {
    public String getStorageClassName(String ourName);

    public String getInternalClassName(String storageClassName);
}
