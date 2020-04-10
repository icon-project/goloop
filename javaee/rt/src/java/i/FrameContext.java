package i;

public interface FrameContext {
    IDBStorage getDBStorage();
    boolean waitForRefund();
}
