package foundation.icon.ee.io;

import i.RuntimeAssertionError;
import org.msgpack.core.MessageBufferPacker;
import org.msgpack.core.MessagePack;

import java.io.IOException;
import java.math.BigInteger;

public class MessagePackDataWriter implements DataWriter {
    private MessageBufferPacker packer;

    public MessagePackDataWriter() {
        this.packer = MessagePack.newDefaultBufferPacker();
    }

    public void write(boolean v) {
        try {
            packer.packBoolean(v);
        } catch (IOException e) {
            RuntimeAssertionError.unexpected(e);
        }
    }

    public void write(byte v) {
        try {
            packer.packByte(v);
        } catch (IOException e) {
            RuntimeAssertionError.unexpected(e);
        }
    }

    public void write(short v) {
        try {
            packer.packShort(v);
        } catch (IOException e) {
            RuntimeAssertionError.unexpected(e);
        }
    }

    public void write(char v) {
        try {
            packer.packInt(v);
        } catch (IOException e) {
            RuntimeAssertionError.unexpected(e);
        }
    }

    public void write(int v) {
        try {
            packer.packInt(v);
        } catch (IOException e) {
            RuntimeAssertionError.unexpected(e);
        }
    }

    public void write(float v) {
        try {
            packer.packFloat(v);
        } catch (IOException e) {
            RuntimeAssertionError.unexpected(e);
        }
    }

    public void write(long v) {
        try {
            packer.packLong(v);
        } catch (IOException e) {
            RuntimeAssertionError.unexpected(e);
        }
    }

    public void write(double v) {
        try {
            packer.packDouble(v);
        } catch (IOException e) {
            RuntimeAssertionError.unexpected(e);
        }
    }

    public void write(BigInteger v) {
        try {
            packer.packBigInteger(v);
        } catch (IOException e) {
            RuntimeAssertionError.unexpected(e);
        }
    }

    public void write(String v) {
        try {
            packer.packString(v);
        } catch (IOException e) {
            RuntimeAssertionError.unexpected(e);
        }
    }

    public void write(byte[] v) {
        try {
            packer.packBinaryHeader(v.length);
            packer.writePayload(v);
        } catch (IOException e) {
            RuntimeAssertionError.unexpected(e);
        }
    }

    public void writeNullity(boolean nullity) {
        if (nullity) {
            writeNull();
        }
    }

    public void writeListHeader(int l) {
        try {
            packer.packArrayHeader(l);
        } catch (IOException e) {
            RuntimeAssertionError.unexpected(e);
        }
    }

    public void writeMapHeader(int l) {
        try {
            packer.packMapHeader(l);
        } catch (IOException e) {
            RuntimeAssertionError.unexpected(e);
        }
    }

    public void writeFooter() {
    }

    private void writeNull() {
        try {
            packer.packNil();
        } catch (IOException e) {
            RuntimeAssertionError.unexpected(e);
        }
    }

    public void flush() {
        try {
            packer.flush();
        } catch (IOException e) {
            RuntimeAssertionError.unexpected(e);
        }
    }

    public byte[] toByteArray() {
        flush();
        return packer.toByteArray();
    }

    public long getTotalWrittenBytes() {
        return packer.getTotalWrittenBytes();
    }
}
