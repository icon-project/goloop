package i;

import org.aion.avm.core.IExternalState;

public interface FrameContext {
    IExternalState getExternalState();
    IDBStorage getDBStorage();
}
