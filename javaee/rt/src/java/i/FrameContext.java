package i;

import org.aion.avm.core.IExternalState;

public interface FrameContext {
    IDBStorage getDBStorage();
    IExternalState getExternalState();
    boolean waitForRefund();
    void limitPendingRefundLength();
    void setStatusFlag(int rerun);
    int getStatusFlag();
    boolean isDeployFrame();
}
