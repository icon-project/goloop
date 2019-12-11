/*
 * Copyright 2019 ICON Foundation
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package foundation.icon.ee.ipc;

import java.io.InputStream;
import java.io.IOException;
import java.io.File;

/*
 * Simple UNIX domain socket implementation for communicating with the Service Manager.
 */
public class Client implements Connection {
    private String path;
    private int fd;

    public Client(String path) {
        this.path = path;
    }

    /**
     * Attach to the target
     *
     * @param path  the UNIX domain socket address for connection
     */
    public static Client connect(String path) throws IOException {
        if (path == null) {
            throw new NullPointerException("path cannot be null");
        }
        Client client = new Client(path);
        client.initialize();
        return client;
    }

    private void initialize() throws IOException {
        File f = new File(path);
        if (!f.exists()) {
            throw new IOException("Unable to open socket file(" + path + ")");
        }

        // Connect to the target
        fd = socket();
        try {
            connect(fd, path);
        } catch (IOException e) {
            close(fd);
            throw e;
        }
    }

    /**
     * Detach from the target
     */
    public void close() throws IOException {
        synchronized (this) {
            close(fd);
            if (this.path != null) {
                this.path = null;
            }
        }
    }

    public void send(byte[] b) throws IOException {
        write(fd, b, 0, b.length);
    }

    public int recv(byte[] b) throws IOException {
        return read(fd, b, 0, b.length);
    }

    public InputStream getInputStream() {
        return new SocketInputStream(fd);
    }

    /*
     * InputStream for the socket connection to get the target
     */
    private static class SocketInputStream extends InputStream {
        int s;

        SocketInputStream(int s) {
            this.s = s;
        }

        public synchronized int read() throws IOException {
            byte[] b = new byte[1];
            int n = this.read(b, 0, 1);
            if (n == 1) {
                return b[0] & 0xff;
            } else {
                return -1;
            }
        }

        public synchronized int read(byte[] bs, int off, int len) throws IOException {
            if ((off < 0) || (off > bs.length) || (len < 0) ||
                ((off + len) > bs.length) || ((off + len) < 0)) {
                throw new IndexOutOfBoundsException();
            } else if (len == 0)
                return 0;

            return Client.read(s, bs, off, len);
        }

        public void close() throws IOException {
            //Client.close(s);
        }
    }

    //-- native methods

    static native int socket() throws IOException;

    static native void connect(int fd, String path) throws IOException;

    static native void close(int fd) throws IOException;

    static native int read(int fd, byte[] buf, int off, int bufLen) throws IOException;

    static native void write(int fd, byte[] buf, int off, int bufLen) throws IOException;

    static {
        System.loadLibrary("client");
    }

    static public Connector connector = Client::connect;
}
