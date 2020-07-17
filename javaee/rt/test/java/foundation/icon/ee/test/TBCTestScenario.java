package foundation.icon.ee.test;

import java.nio.ByteBuffer;
import java.nio.charset.StandardCharsets;
import java.util.ArrayList;
import java.util.Arrays;

/**
 *  Represents TBC test scenario. The scenario can be compiled into TBC.
 */
public class TBCTestScenario {
    private static final int MAX_CODE = 8 * 1024;
    private ByteBuffer bb = ByteBuffer.allocate(MAX_CODE);
    private ArrayList<Integer> lengthOffsets = new ArrayList<>();
    private int expectCount = 0;

    public byte[] compile() {
        if (!lengthOffsets.isEmpty()) {
            endCall();
        }
        return Arrays.copyOf(bb.array(), bb.position());
    }

    public int getExpectCount() {
        return expectCount;
    }

    public TBCTestScenario call(byte[] addr) {
        bb.put(TBCProtocol.CALL);
        bb.put(addr);
        lengthOffsets.add(bb.position());
        bb.putShort((short) 0);
        return this;
    }

    private void endCall() {
        int offset = lengthOffsets.remove(lengthOffsets.size() - 1);
        int len = bb.position() - offset - 2;
        bb.putShort(offset, (short) (len & 0xffff));
    }

    public TBCTestScenario ret() {
        endCall();
        return this;
    }

    public TBCTestScenario revert() {
        return revert(0);
    }

    public TBCTestScenario revert(int code) {
        bb.put(TBCProtocol.REVERT);
        bb.putShort((short) (code & 0xffff));
        endCall();
        return this;
    }

    public TBCTestScenario set(int type, int id, String s) {
        bb.put(TBCProtocol.SET);
        bb.put((byte)(type & 0xff));
        bb.put((byte)(id & 0xff));
        if (s == null) {
            bb.putShort((short)-1);
        } else {
            bb.putShort((short) (s.length() & 0xffff));
            bb.put(s.getBytes(StandardCharsets.UTF_8));
        }
        return this;
    }

    public TBCTestScenario append(int type, int id, String s) {
        bb.put(TBCProtocol.APPEND);
        bb.put((byte)(type & 0xff));
        bb.put((byte)(id & 0xff));
        bb.putShort((short) (s.length() & 0xffff));
        bb.put(s.getBytes(StandardCharsets.UTF_8));
        return this;
    }

    public TBCTestScenario expect(int type, int id, String s, byte op) {
        ++expectCount;
        bb.put(TBCProtocol.EXPECT);
        bb.put((byte)(type & 0xff));
        bb.put((byte)(id & 0xff));
        if (s == null) {
            bb.putShort((short)-1);
        } else {
            bb.putShort((short) (s.length() & 0xffff));
            bb.put(s.getBytes(StandardCharsets.UTF_8));
        }
        bb.put(op);
        return this;
    }

    public TBCTestScenario expect(int type, int id, String s) {
        return expect(type, id, s, TBCProtocol.CMP_EQ);
    }

    public TBCTestScenario expectNE(int type, int id, String s) {
        return expect(type, id, s, TBCProtocol.CMP_NE);
    }

    public TBCTestScenario setRef(int type, int id, int type2, int id2) {
        bb.put(TBCProtocol.SET_REF);
        bb.put((byte)(type & 0xff));
        bb.put((byte)(id & 0xff));
        bb.put((byte)(type2 & 0xff));
        bb.put((byte)(id2 & 0xff));
        return this;
    }

    public TBCTestScenario expectRef(int type, int id, int type2, int id2, byte op) {
        ++expectCount;
        bb.put(TBCProtocol.EXPECT_REF);
        bb.put((byte)(type & 0xff));
        bb.put((byte)(id & 0xff));
        bb.put((byte)(type2 & 0xff));
        bb.put((byte)(id2 & 0xff));
        bb.put(op);
        return this;
    }

    public TBCTestScenario expectRef(int type, int id, int type2, int id2) {
        return expectRef(type, id, type2, id2, TBCProtocol.CMP_EQ);
    }

    public TBCTestScenario expectRefNE(int type, int id, int type2, int id2) {
        return expectRef(type, id, type2, id2, TBCProtocol.CMP_NE);
    }
}
