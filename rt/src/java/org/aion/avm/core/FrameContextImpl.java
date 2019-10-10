package org.aion.avm.core;

import i.FrameContext;
import i.IBlockchainRuntime;
import i.InternedClasses;
import org.aion.avm.core.persistence.LoadedDApp;

public class FrameContextImpl implements FrameContext {
    private LoadedDApp dapp;
    private InternedClasses icm;
    private IBlockchainRuntime br;

    FrameContextImpl(LoadedDApp dapp, InternedClasses icm, IBlockchainRuntime br) {
        this.dapp = dapp;
        this.icm = icm;
        this.br = br;
    }

    public Object deserializeObject(byte[] rawGraphData) {
        return dapp.deserializeObject(icm, rawGraphData);
    }

    public byte[] serializeObject(Object v) {
        return dapp.serializeObject(v);
    }

    public IBlockchainRuntime getBlockchainRuntime() {
        return br;
    }
}
