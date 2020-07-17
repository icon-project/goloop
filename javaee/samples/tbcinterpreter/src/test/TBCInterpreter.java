package test;

import score.Address;
import score.Context;
import score.ScoreRevertException;
import score.annotation.EventLog;
import score.annotation.External;

import java.util.Arrays;

public class TBCInterpreter {
    private static final String[] sVar = new String[TBCProtocol.MAX_VAR];

    private final String name;
    private final String[] iVar = new String[TBCProtocol.MAX_VAR];
    private StringBuilder sb;

    public TBCInterpreter(String _name) {
        this.name = _name;
    }

    @External
    public void runAndLogResult(byte[] _code) {
        String res = run(_code);
        Context.println(res);
        Event_(res);
    }

    @External
    public String run(byte[] _code) {
        var old = sb;
        sb = new StringBuilder();
        Event("Enter: " + name);
        try {
            doRunImpl(_code);
            Event("Exit by Return: " + name);
        } catch (Throwable t) {
            Event("Exit by Exception: " + name + " e="+t);
        }
        var res = sb.toString();
        sb = old;
        return res;
    }

    @EventLog(indexed=1)
    private void Event_(String eventData) {
    }

    private void Event(String eventData) {
        if (sb.length() > 0) {
            sb.append('\n');
        }
        sb.append(eventData);
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

    private void doRunImpl(byte[] code) {
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
                    int ccodeLen = (code[offset++] << 8) & 0xff | (code[offset++] & 0xff);
                    var ccode = Arrays.copyOfRange(code, offset, offset + ccodeLen);
                    offset += ccodeLen;
                    String res = (String)Context.call(addr, "run",
                            (Object) ccode);
                    Event(res);
                } catch (ScoreRevertException e) {
                    Event(e.getMessage());
                }
            } else if (insn == TBCProtocol.REVERT) {
                int rcode = (code[offset++] << 8) & 0xff | (code[offset++] & 0xff);
                Event("Exit by Revert: " + name);
                Context.revert(rcode, sb.toString());
            } else if (insn == TBCProtocol.SET) {
                int t = code[offset++] & 0xff;
                int id = code[offset++] & 0xff;
                int len = (code[offset++] << 8) & 0xff | (code[offset++] & 0xff);
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
                int len = (code[offset++] << 8) & 0xff | (code[offset++] & 0xff);
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
                int len = (code[offset++] << 8) & 0xff | (code[offset++] & 0xff);
                String val = null;
                if (len >= 0) {
                    val = new String(code, offset, len);
                }
                offset += len;
                String xvar = getRef(t, id, lVar);
                boolean eqRes = code[offset++] == TBCProtocol.CMP_EQ;
                if (val.equals(xvar) == eqRes) {
                    Event("EXPECT [OK] xvar=" + xvar);
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
                } else {
                    Event("EXPECT_REF [ERROR]");
                }
            } else {
                Event("Unexpected insn " + insn);
            }
        }
    }
}

