package foundation.icon.ee;

import foundation.icon.ee.test.ContractAddress;
import foundation.icon.ee.test.GoldenTest;
import org.junit.jupiter.api.Test;
import score.Address;
import score.Context;
import score.annotation.External;

import java.nio.ByteBuffer;
import java.nio.charset.StandardCharsets;
import java.util.ArrayList;
import java.util.Arrays;

public class ScenarioTest extends GoldenTest {
    public static class Score {
        // short addr
        // short codeLen
        // byte[] code
        public static final byte CALL = 0;

        public static final byte REVERT = 1;

        // short len
        // byte[] string
        public static final byte SET_SVAR = 2;

        // short len
        // byte[] string
        public static final byte ADD_TO_SVAR = 3;

        // short len
        // byte[] string
        public static final byte EXPECT_SVAR = 4;

        private String sVar;

        @External
        public void run(byte[] code) {
            var ba = Context.getAddress().toByteArray();
            int addr = ((ba[1] & 0xff) << 8) | (ba[2] & 0xff);
            Context.println("Enter addr=" + addr);
            try {
                doRunImpl(code);
                Context.println("Exit by Return addr=" + addr);
            } catch(Throwable t) {
                Context.println("Exit by Throwable addr=" + addr + " t=" + t);
            }
        }

        private void doRunImpl(byte[] code) {
            int offset = 0;
            while (offset < code.length) {
                int insn = code[offset++] & 0xff;
                if (insn == CALL) {
                    try {
                        var ba = new byte[21];
                        ba[0] = 1;
                        ba[1] = code[offset++];
                        ba[2] = code[offset++];
                        var addr = new Address(ba);
                        int ccodeLen = ((code[offset++] & 0xff) << 8) | (code[offset++] & 0xff);
                        var ccode = Arrays.copyOfRange(code, offset, offset + ccodeLen);
                        offset += ccodeLen;
                        Context.call(addr, "run", (Object) ccode);
                    } catch (Exception e) {
                        Context.println("Exception e=" + e);
                    }
                } else if (insn == REVERT){
                    var ba = Context.getAddress().toByteArray();
                    int addr = ((ba[1] & 0xff) << 8) | (ba[2] & 0xff);
                    Context.println("Exit by Revert addr=" + addr);
                    Context.revert();
                } else if (insn == SET_SVAR) {
                    int len = ((code[offset++] & 0xff) << 8) | (code[offset++] & 0xff);
                    var s = new String(code, offset, len);
                    offset += len;
                    sVar = s;
                    Context.println("Set sVar=" + sVar);
                } else if (insn == ADD_TO_SVAR) {
                    int len = ((code[offset++] & 0xff) << 8) | (code[offset++] & 0xff);
                    var s = new String(code, offset, len);
                    offset += len;
                    var before = sVar;
                    sVar += s;
                    Context.println("AddTo sVar=" + before + " s=" + s + " => sVar=" + sVar);
                } else if (insn == EXPECT_SVAR) {
                    int len = ((code[offset++] & 0xff) << 8) | (code[offset++] & 0xff);
                    var s = new String(code, offset, len);
                    offset += len;
                    if (s.equals(sVar)) {
                        Context.println("Expect [OK] expected sVar=" + sVar);
                    } else {
                        Context.println("Expect [ERROR] expected sVar=" + s + " observed sVar=" + sVar);
                    }
                }
            }
        }
    }

    public static class Compiler {
        private static final int MAX_CODE = 8 * 1024;
        private ByteBuffer bb = ByteBuffer.allocate(MAX_CODE);
        private ArrayList<Integer> lengthOffsets = new ArrayList<>();

        public byte[] compile() {
            return Arrays.copyOf(bb.array(), bb.position());
        }

        public Compiler call(ContractAddress c) {
            return call(c.getAddress());
        }

        public Compiler call(foundation.icon.ee.types.Address addr) {
            bb.put(Score.CALL);
            var ba = addr.toByteArray();
            bb.put(ba[1]);
            bb.put(ba[2]);
            lengthOffsets.add(bb.position());
            bb.putShort((short) 0);
            return this;
        }

        private void endCall() {
            int offset = lengthOffsets.remove(lengthOffsets.size() - 1);
            int len = bb.position() - offset - 2;
            bb.putShort(offset, (short) (len & 0xffff));
        }

        public Compiler ret() {
            endCall();
            return this;
        }

        public Compiler revert() {
            bb.put(Score.REVERT);
            endCall();
            return this;
        }

        public Compiler setSVar(String s) {
            bb.put(Score.SET_SVAR);
            bb.putShort((short) (s.length() & 0xffff));
            bb.put(s.getBytes(StandardCharsets.UTF_8));
            return this;
        }

        public Compiler addToSVar(String s) {
            bb.put(Score.ADD_TO_SVAR);
            bb.putShort((short) (s.length() & 0xffff));
            bb.put(s.getBytes(StandardCharsets.UTF_8));
            return this;
        }

        public Compiler expectSVar(String s) {
            bb.put(Score.EXPECT_SVAR);
            bb.putShort((short) (s.length() & 0xffff));
            bb.put(s.getBytes(StandardCharsets.UTF_8));
            return this;
        }
    }

    @Test
    public void testBasic() {
        var c1 = sm.mustDeploy(Score.class);
        var c2 = sm.mustDeploy(Score.class);
        var c3 = sm.mustDeploy(Score.class);
        var code = new Compiler()
                .call(c2)
                .ret()
                .call(c3)
                    .call(c2)
                    .ret()
                .revert()
                .compile();
        c1.invoke("run", (Object)code);
    }

    @Test
    public void testIndirectRecursion() {
        var c1 = sm.mustDeploy(Score.class);
        var c2 = sm.mustDeploy(Score.class);
        var c3 = sm.mustDeploy(Score.class);
        c1.invoke("run", (Object)new Compiler()
                .call(c2)
                    .setSVar("")
                .ret()
                .call(c2)
                    .addToSVar("a")
                .revert()
                .call(c2)
                    .addToSVar("b")
                .ret()
                .call(c3)
                    .call(c2)
                        .addToSVar("c")
                        .expectSVar("bc")
                    .revert()
                .revert()
                .call(c3)
                    .call(c2)
                        .addToSVar("d")
                    .ret()
                .revert()
                .call(c3)
                    .call(c2)
                        .addToSVar("e")
                    .revert()
                .ret()
                .call(c3)
                    .call(c2)
                        .addToSVar("f")
                    .ret()
                .ret()
                .call(c2)
                    .expectSVar("bf")
                .ret()
                .compile()
        );
    }

    @Test
    public void testDirectRecursion() {
        var c1 = sm.mustDeploy(Score.class);
        c1.invoke("run", (Object)new Compiler()
                .setSVar("")
                .call(c1)
                    .addToSVar("a")
                .revert()
                .expectSVar("")
                .call(c1)
                    .addToSVar("b")
                .ret()
                .expectSVar("b")
                .call(c1)
                    .call(c1)
                        .addToSVar("c")
                        .expectSVar("bc")
                    .revert()
                    .expectSVar("b")
                .revert()
                .call(c1)
                    .call(c1)
                        .addToSVar("d")
                        .expectSVar("bd")
                    .ret()
                    .expectSVar("bd")
                .revert()
                .expectSVar("b")
                .call(c1)
                    .call(c1)
                        .addToSVar("e")
                        .expectSVar("be")
                    .revert()
                    .expectSVar("b")
                .ret()
                .expectSVar("b")
                .call(c1)
                    .call(c1)
                        .addToSVar("f")
                        .expectSVar("bf")
                    .ret()
                    .expectSVar("bf")
                .ret()
                .expectSVar("bf")
                .compile()
        );
    }

    @Test
    public void testDirectRecursion2() {
        var c1 = sm.mustDeploy(Score.class);
        c1.invoke("run", (Object)new Compiler()
                .setSVar("1")
                .call(c1)
                    .expectSVar("1")
                    .setSVar("2")
                .ret()
                .expectSVar("2")
                .call(c1)
                    .expectSVar("2")
                    .setSVar("3")
                .revert()
                .expectSVar("2")
                .call(c1)
                    .expectSVar("2")
                    .setSVar("4")
                .ret()
                .expectSVar("4")
                .compile()
        );
    }

    @Test
    public void testIndirectRecursion2() {
        var c1 = sm.mustDeploy(Score.class);
        var c2 = sm.mustDeploy(Score.class);
        c1.invoke("run", (Object)new Compiler()
                .setSVar("1")
                .call(c2)
                    .call(c1)
                        .expectSVar("1")
                        .setSVar("2")
                    .ret()
                    .call(c1)
                        .expectSVar("2")
                        .setSVar("3")
                    .revert()
                    .call(c1)
                        .expectSVar("2")
                        .setSVar("4")
                    .ret()
                .ret()
                .expectSVar("4")
                .compile()
        );
    }

    @Test
    public void testIndirectRecursionWithMultiEE() {
        createAndAcceptNewJAVAEE();
        var c1 = sm.mustDeploy(Score.class);
        sm.setIndexer((addr) -> 1);
        var c2 = sm.mustDeploy(Score.class);
        sm.setIndexer((addr) -> {
            if (addr.equals(c1.getAddress())) {
                return 0;
            }
            return 1;
        });
        c1.invoke("run", (Object)new Compiler()
                .setSVar("1")
                .call(c2)
                    .call(c1)
                        .expectSVar("1")
                        .setSVar("2")
                    .ret()
                    .call(c1)
                        .expectSVar("2")
                        .setSVar("3")
                    .revert()
                    .call(c1)
                        .expectSVar("2")
                        .setSVar("4")
                    .ret()
                .revert()
                .call(c2)
                    .call(c1)
                        .expectSVar("1")
                        .setSVar("2")
                    .ret()
                    .call(c1)
                        .expectSVar("2")
                        .setSVar("3")
                    .revert()
                    .call(c1)
                        .expectSVar("2")
                        .setSVar("4")
                    .ret()
                .ret()
                .expectSVar("4")
                .compile()
        );
    }
}
