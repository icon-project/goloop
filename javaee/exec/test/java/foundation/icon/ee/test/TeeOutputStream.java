package foundation.icon.ee.test;

import java.io.IOException;
import java.io.OutputStream;

public class TeeOutputStream extends OutputStream {
    private final OutputStream o1, o2;

    public TeeOutputStream(OutputStream o1, OutputStream o2) {
        this.o1 = o1;
        this.o2 = o2;
    }

    public void write(int b) throws IOException {
        o1.write(b);
        o2.write(b);
    }

    public void write(byte[] b) throws IOException {
        o1.write(b);
        o2.write(b);
    }

    public void write(byte[] b, int off, int len) throws IOException {
        o1.write(b, off, len);
        o2.write(b, off, len);
    }

    public void flush() throws IOException {
        o1.flush();
        o2.flush();
    }

    public void close() throws IOException {
        try {
            o1.close();
        } finally {
            o2.close();
        }
    }
}
