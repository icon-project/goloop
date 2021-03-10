package test;

import score.Address;
import score.Context;
import score.UserRevertedException;
import score.annotation.EventLog;
import score.annotation.External;

import java.math.BigInteger;
import java.util.Arrays;

public class TBCInterpreter {
    private static final String[] sVar = new String[TBCProtocol.MAX_VAR];

    private final String name;
    private final String[] iVar = new String[TBCProtocol.MAX_VAR];

    public TBCInterpreter(String _name) {
        this.name = _name;
    }

    @External
    public void runAndLogResult(byte[] _code) {
        int res = run(_code);
        Event_(res);
    }

    @External
    public int run(byte[] _code) {
        Event("Enter: " + name);
        int res = 0;
        try {
            res = doRunImpl(_code);
            Event("Exit by Return: " + name);
        } catch (Throwable t) {
            t.printStackTrace();
            Event("Exit by Exception: " + name + " e="+t);
            Context.require(false);
        }
        return res;
    }

    @EventLog(indexed=1)
    private void Event_(int eventData) {
    }

    private void Event(String eventData) {
        Context.println(eventData);
    }

    private String getRef(int type, int id, String[] lVar) {
        switch (type) {
            case TBCProtocol.VAR_TYPE_STATIC:
                return sVar[id];
            case TBCProtocol.VAR_TYPE_INSTANCE:
                return iVar[id];
            case TBCProtocol.VAR_TYPE_LOCAL:
                return lVar[id];
            default:
                Event("Unexpected var type " + type);
                return null;
        }
    }

    private static boolean equals(Object a, Object b) {
        return (a == b) || (a != null && a.equals(b));
    }

    private int doRunImpl(byte[] code) {
        int okCount = 0;
        int offset = 0;
        final String[] lVar = new String[TBCProtocol.MAX_VAR];
        while (offset < code.length) {

            int insn = code[offset++] & 0xff;
            if (insn == TBCProtocol.CALL) {
                try {
                    var addrBytes = Arrays.copyOfRange(code, offset,
                            offset+Address.LENGTH);
                    offset += Address.LENGTH;
                    var addr = new Address(addrBytes);
                    int ccodeLen = ((code[offset++] & 0xff) << 8) | (code[offset++] & 0xff);
                    var ccode = Arrays.copyOfRange(code, offset, offset + ccodeLen);
                    offset += ccodeLen;
                    BigInteger res = (BigInteger)Context.call(addr, "run",
                            (Object) ccode);
                    assert res != null;
                    okCount += res.intValue();
                } catch (UserRevertedException e) {
                    okCount += e.getCode();
                }
            } else if (insn == TBCProtocol.REVERT) {
                int rcode = ((code[offset++] & 0xff) << 8) | (code[offset++] & 0xff);
                Event("Exit by Revert: " + name);
                Context.revert(okCount);
            } else if (insn == TBCProtocol.SET) {
                int t = code[offset++] & 0xff;
                int id = code[offset++] & 0xff;
                short len = (short)(((code[offset++]) << 8) | (code[offset++] & 0xff));
                String val = null;
                if (len >= 0) {
                    val = new String(code, offset, len);
                }
                offset += len;
                switch (t) {
                    case TBCProtocol.VAR_TYPE_STATIC:
                        sVar[id] = val;
                        break;
                    case TBCProtocol.VAR_TYPE_INSTANCE:
                        iVar[id] = val;
                        break;
                    case TBCProtocol.VAR_TYPE_LOCAL:
                        lVar[id] = val;
                        break;
                    default:
                        Event("Unexpected var type " + t);
                        break;
                }
            } else if (insn == TBCProtocol.APPEND) {
                int t = code[offset++] & 0xff;
                int id = code[offset++] & 0xff;
                int len = ((code[offset++] & 0xff) << 8) | (code[offset++] & 0xff);
                var val = new String(code, offset, len);
                offset += len;
                switch (t) {
                    case TBCProtocol.VAR_TYPE_STATIC:
                        sVar[id] += val;
                        break;
                    case TBCProtocol.VAR_TYPE_INSTANCE:
                        iVar[id] += val;
                        break;
                    case TBCProtocol.VAR_TYPE_LOCAL:
                        lVar[id] += val;
                        break;
                    default:
                        Event("Unexpected var type " + t);
                        break;
                }
            } else if (insn == TBCProtocol.EXPECT) {
                int t = code[offset++] & 0xff;
                int id = code[offset++] & 0xff;
                short len = (short)((code[offset++] << 8) | (code[offset++] & 0xff));
                String val = null;
                if (len >= 0) {
                    val = new String(code, offset, len);
                }
                offset += len;
                String xvar = getRef(t, id, lVar);
                boolean eqRes = code[offset++] == TBCProtocol.CMP_EQ;
                if (equals(val, xvar) == eqRes) {
                    Event("EXPECT [OK] xvar=" + xvar);
                    ++okCount;
                } else {
                    Event("EXPECT [ERROR] expected=" + val +
                            " observed=" + xvar);
                }
            } else if (insn == TBCProtocol.SET_REF) {
                int t = code[offset++] & 0xff;
                int id = code[offset++] & 0xff;
                int t2 = code[offset++] & 0xff;
                int id2 = code[offset++] & 0xff;
                String rhs = getRef(t2, id2, lVar);
                switch (t) {
                    case TBCProtocol.VAR_TYPE_STATIC:
                        sVar[id] = rhs;
                        break;
                    case TBCProtocol.VAR_TYPE_INSTANCE:
                        iVar[id] = rhs;
                        break;
                    case TBCProtocol.VAR_TYPE_LOCAL:
                        lVar[id] = rhs;
                        break;
                    default:
                        Event("Unexpected var type " + t);
                        break;
                }
            } else if (insn == TBCProtocol.EXPECT_REF) {
                int t = code[offset++] & 0xff;
                int id = code[offset++] & 0xff;
                int t2 = code[offset++] & 0xff;
                int id2 = code[offset++] & 0xff;
                String lhs = getRef(t, id, lVar);
                String rhs = getRef(t2, id2, lVar);
                boolean eqRes = code[offset++] == TBCProtocol.CMP_EQ;
                if ((lhs == rhs) == eqRes) {
                    Event("EXPECT_REF [OK]");
                    ++okCount;
                } else {
                    Event("EXPECT_REF [ERROR]");
                }
            } else {
                Event("Unexpected insn " + insn);
            }
        }
        return okCount;
    }
}
