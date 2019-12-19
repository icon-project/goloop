package org.aion.avm.core;

import i.FrameContext;
import i.IBlockchainRuntime;
import i.IDBStorage;
import i.InternedClasses;
import org.aion.avm.core.persistence.LoadedDApp;

public class FrameContextImpl implements FrameContext {
    private IDBStorage dbs;
    private IBlockchainRuntime br;

    FrameContextImpl(IExternalState ctx, LoadedDApp dapp, InternedClasses icm, IBlockchainRuntime br) {
        this.dbs = new DBStorage(ctx, dapp, icm );
        this.br = br;
    }

    public IBlockchainRuntime getBlockchainRuntime() {
        return br;
    }

    public IDBStorage getDBStorage() {
        return dbs;
    }
}
