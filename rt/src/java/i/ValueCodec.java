package i;

public class ValueCodec {
    public static byte[] encodeValue(Object v) {
        var fc = IInstrumentation.attachedThreadInstrumentation.get().getFrameContext();
        return fc.serializeObject(v);
    }

    public static Object decodeValue(byte[] raw) {
        var fc = IInstrumentation.attachedThreadInstrumentation.get().getFrameContext();
        return fc.deserializeObject(raw);
    }
}
