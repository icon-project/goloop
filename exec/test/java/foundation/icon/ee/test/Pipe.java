package foundation.icon.ee.test;

import foundation.icon.ee.ipc.Connection;
import foundation.icon.ee.ipc.Connector;

import java.io.IOException;
import java.io.InputStream;
import java.io.PipedInputStream;
import java.io.PipedOutputStream;

public class Pipe implements Connection {
    private static final int bufSize = 400*1024;
    private PipedInputStream in;
    private PipedOutputStream out;

    private Pipe() {
        this(new PipedInputStream(bufSize), new PipedOutputStream());
    }

    private Pipe(PipedInputStream in, PipedOutputStream out) {
        this.in = in;
        this.out = out;
    }

    public static Pipe[] createPair() {
        try {
            var p = new Pipe();
            var p2 = new Pipe(
                    new PipedInputStream(p.out, bufSize),
                    new PipedOutputStream(p.in)
            );
            return new Pipe[] { p, p2 };
        } catch (Exception e) {
            throw new AssertionError(e);
        }
    }

    public static Object[] createPipeAndConnector() {
        var pair = createPair();
        var conn = new Connector() {
            public Connection connect(String addr) {
                return pair[1];
            }
        };
        return new Object[] { pair[0], conn };
    }

    public InputStream getInputStream() {
        return in;
    }

    public void close() throws IOException {
        out.close();
        //in.close();
    }

    public void send(byte[] data) throws IOException {
        out.write(data);
        out.flush();
    }
}
