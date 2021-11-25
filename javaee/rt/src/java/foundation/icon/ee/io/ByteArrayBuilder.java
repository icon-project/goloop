package foundation.icon.ee.io;

import java.io.OutputStream;
import java.util.Arrays;

// synchronized-free
// remove exception spec
// unsafe buffer peek
// size
public class ByteArrayBuilder extends OutputStream {
    private static final int INITIAL_CAP = 8;
    private byte[] buf = new byte[INITIAL_CAP];
    private int size;

    private void ensureCap(int req) {
        if (req > buf.length) {
            int newCap = buf.length * 2;
            if (newCap < req) {
                newCap = req;
            }
            buf = Arrays.copyOf(buf, newCap);
        }
    }

    public void write(int b) {
        ensureCap(size + 1);
        buf[size++] = (byte) b;
    }

    public void write(byte[] b) {
        write(b, 0, b.length);
    }

    public void write(byte[] b, int off, int len) {
        ensureCap(size + len);
        System.arraycopy(b, off, buf, size, len);
        size += len;
    }

    public void flush() {
    }

    public void close() {
    }

    public byte[] array() {
        return buf;
    }

    public int size() {
        return size;
    }

    public void resize(int size) {
        this.size = size;
    }
}
