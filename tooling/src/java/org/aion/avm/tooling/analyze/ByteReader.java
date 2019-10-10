package org.aion.avm.tooling.analyze;

import java.nio.ByteBuffer;

public class ByteReader {

    private ByteBuffer buffer;

    public ByteReader(byte[] bytes) {
        buffer = ByteBuffer.wrap(bytes);
    }

    // The types u1, u2, and u4 represent an unsigned one-, two-, or four-byte quantity, respectively
    // ByteBuffer is read in a big-endian byte order
    public byte readU1() {
        return buffer.get();
    }

    public short readU2() {
        return buffer.getShort();
    }

    public int readU4() {
        return buffer.getInt();
    }

    public byte[] readNBytes(int n) {
        byte[] utf8 = new byte[n];
        buffer.get(utf8);
        return utf8;
    }

    public int position() {
        return buffer.position();
    }
}
