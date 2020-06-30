package foundation.icon.ee.io;

import org.msgpack.core.MessageFormat;
import org.msgpack.core.MessageInsufficientBufferException;
import org.msgpack.core.MessagePack;
import org.msgpack.core.MessageTypeException;
import org.msgpack.core.MessageUnpacker;
import org.msgpack.value.ValueType;

import java.io.IOException;
import java.math.BigInteger;
import java.util.ArrayList;

public class MessagePackDataReader implements DataReader {
    private static class ListFrame {
        int current;
        int length;
    }

    private MessageUnpacker unpacker;
    private ArrayList<ListFrame> frames = new ArrayList<>();
    private ListFrame topFrame;

    public MessagePackDataReader(byte[] data) {
        this(MessagePack.newDefaultUnpacker(data));
    }

    public MessagePackDataReader(MessageUnpacker unpacker) {
        this.unpacker = unpacker;
        this.topFrame = new ListFrame();
        this.frames.add(topFrame);
        this.topFrame.length = Integer.MAX_VALUE;
    }

    public RuntimeException convert(Exception e) {
        if (e instanceof MessageTypeException ||
                e instanceof MessageInsufficientBufferException) {
            return new IllegalStateException();
        }
        return new UnsupportedOperationException();
    }

    public boolean readBoolean() {
        try {
            var res = unpacker.unpackBoolean();
            topFrame.current++;
            return res;
        } catch (Exception e) {
            throw convert(e);
        }
    }

    public byte readByte() {
        try {
            var res = unpacker.unpackByte();
            topFrame.current++;
            return res;
        } catch (Exception e) {
            throw convert(e);
        }
    }

    public short readShort() {
        try {
            var res = unpacker.unpackShort();
            topFrame.current++;
            return res;
        } catch (Exception e) {
            throw convert(e);
        }
    }

    public char readChar() {
        try {
            var res = (char) unpacker.unpackInt();
            topFrame.current++;
            return res;
        } catch (Exception e) {
            throw convert(e);
        }
    }

    public int readInt() {
        try {
            var res = unpacker.unpackInt();
            topFrame.current++;
            return res;
        } catch (Exception e) {
            throw convert(e);
        }
    }

    public float readFloat() {
        try {
            var res = unpacker.unpackFloat();
            topFrame.current++;
            return res;
        } catch (Exception e) {
            throw convert(e);
        }
    }

    public long readLong() {
        try {
            var res = unpacker.unpackLong();
            topFrame.current++;
            return res;
        } catch (Exception e) {
            throw convert(e);
        }
    }

    public double readDouble() {
        try {
            var res = unpacker.unpackDouble();
            topFrame.current++;
            return res;
        } catch (Exception e) {
            throw convert(e);
        }
    }

    public BigInteger readBigInteger() {
        try {
            var res = unpacker.unpackBigInteger();
            topFrame.current++;
            return res;
        } catch (Exception e) {
            throw convert(e);
        }
    }

    public String readString() {
        try {
            var res = unpacker.unpackString();
            topFrame.current++;
            return res;
        } catch (Exception e) {
            throw convert(e);
        }
    }

    public byte[] readByteArray() {
        try {
            MessageFormat fmt = unpacker.getNextFormat();
            if (fmt.getValueType() != ValueType.BINARY) {
                throw new IllegalStateException();
            }
            var l = unpacker.unpackBinaryHeader();
            var ba = new byte[l];
            unpacker.readPayload(ba);
            topFrame.current++;
            return ba;
        } catch (Exception e) {
            throw convert(e);
        }
    }

    public boolean readNullity() {
        return tryReadNull();
    }

    public void skip(int count) {
        try {
            unpacker.skipValue(count);
            topFrame.current += count;
        } catch (Exception e) {
            throw convert(e);
        }
    }

    private void readContainerHeader(ValueType type) {
        try {
            MessageFormat fmt = unpacker.getNextFormat();
            ValueType vt = fmt.getValueType();
            int l;
            if (vt != type) {
                throw new IllegalStateException();
            }
            if (vt == ValueType.ARRAY) {
                l = unpacker.unpackArrayHeader();
            } else {
                assert vt == ValueType.MAP;
                l = unpacker.unpackMapHeader();
            }
            topFrame = new ListFrame();
            topFrame.length = l;
            frames.add(topFrame);
        } catch (Exception e) {
            throw convert(e);
        }
    }

    public void readListHeader() {
        readContainerHeader(ValueType.ARRAY);
    }

    public void readMapHeader() {
        readContainerHeader(ValueType.MAP);
    }

    public boolean hasNext() {
        if (frames.size() == 1) {
            try {
                return unpacker.hasNext();
            } catch (IOException e) {
                throw convert(e);
            }
        }
        return topFrame.current < topFrame.length;
    }

    public void readFooter() {
        frames.remove(frames.size() - 1);
        topFrame = frames.get(frames.size() - 1);
        topFrame.current++;
    }

    private boolean tryReadNull() {
        try {
            MessageFormat fmt = unpacker.getNextFormat();
            if (fmt == MessageFormat.NIL) {
                unpacker.unpackNil();
                topFrame.current++;
                return true;
            }
            return false;
        } catch (Exception e) {
            throw convert(e);
        }
    }

    public long getTotalReadBytes() {
        return unpacker.getTotalReadBytes();
    }
}
