package org.aion.avm.core.persistence;

import org.aion.avm.core.ClassRenamer;
import org.aion.avm.core.ClassRenamer.ArrayType;

public class StandardNameMapper implements IPersistenceNameMapper {
    private final ClassRenamer classRenamer;

    public StandardNameMapper(ClassRenamer classRenamer) {
        this.classRenamer = classRenamer;
    }

    @Override
    public String getStorageClassName(String ourName) {
        return this.classRenamer.toPreRename(ourName);
    }

    @Override
    public String getInternalClassName(String storageClassName) {
        return this.classRenamer.toPostRename(storageClassName, ArrayType.PRECISE_TYPE);
    }
}
