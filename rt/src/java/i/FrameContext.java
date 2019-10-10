package i;

public interface FrameContext {
    Object deserializeObject(byte[] rawGraphData);
    byte[] serializeObject(Object v);
    IBlockchainRuntime getBlockchainRuntime();
}
