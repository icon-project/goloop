package org.aion.avm.core;

import i.FrameContext;
import i.IBlockchainRuntime;
import i.IDBStorage;
import i.InternedClasses;
import org.aion.avm.core.persistence.LoadedDApp;

public class FrameContextImpl implements FrameContext {
    private IDBStorage dbs;

    FrameContextImpl(IExternalState ctx) {
        this.dbs = new DBStorage(ctx);
    }

    public IDBStorage getDBStorage() {
        return dbs;
    }
}
