package org.aion.avm.core;

import i.FrameContext;
import i.IBlockchainRuntime;
import i.IDBStorage;
import i.InternedClasses;
import org.aion.avm.core.persistence.LoadedDApp;

public class FrameContextImpl implements FrameContext {
    private IExternalState externalState;
    private IDBStorage dbs;

    FrameContextImpl(IExternalState externalState) {
        this.externalState = externalState;
        this.dbs = new DBStorage(externalState);
    }

    public IDBStorage getDBStorage() {
        return dbs;
    }

    public boolean waitForRefund() {
        return externalState.waitForCallback();
    }
}
