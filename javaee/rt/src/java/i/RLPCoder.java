package i;

import java.io.ByteArrayOutputStream;
import java.nio.charset.StandardCharsets;

import a.ByteArray;
import s.java.lang.String;
import s.java.lang.Byte;
import s.java.lang.Short;
import s.java.lang.Integer;
import s.java.lang.Long;
import s.java.lang.Character;
import p.score.Address;

import java.math.BigInteger;

public class RLPCoder {
    private ByteArrayOutputStream bos;

    public RLPCoder() {
        bos = new ByteArrayOutputStream();
    }

    static final int SHORT_BASE = 0x80;
    static final int SHORT_LEN_LIMIT = 55;
    static final int LONG_BASE = 0xb7;

    public void write(byte[] bs) {
        bos.write(bs, 0, bs.length);
    }

    public byte[] toByteArray() {
        return bos.toByteArray();
    }

    public void encode(int v) {
        var bs = BigInteger.valueOf(v).toByteArray();
        encode(bs);
    }

    public void encode(Object v) {
        if (v instanceof String) {
            var bs = ((String) v).getUnderlying().getBytes(StandardCharsets.UTF_8);
            encode(bs);
        } else if (v instanceof ByteArray) {
            var bs = ((ByteArray) v).getUnderlying();
            encode(bs);
        } else if (v instanceof Address) {
            var bs = ((Address) v).toByteArray();
            encode(bs);
        } else if (v instanceof s.java.math.BigInteger) {
            var bs = ((s.java.math.BigInteger) v).getUnderlying().toByteArray();
            encode(bs);
        } else if (v instanceof Byte) {
            var vv = ((Byte) v).getUnderlying();
            var bs = BigInteger.valueOf(vv).toByteArray();
            encode(bs);
        } else if (v instanceof Short) {
            var vv = ((Short) v).getUnderlying();
            var bs = BigInteger.valueOf(vv).toByteArray();
            encode(bs);
        } else if (v instanceof Integer) {
            var vv = ((Integer) v).getUnderlying();
            var bs = BigInteger.valueOf(vv).toByteArray();
            encode(bs);
        } else if (v instanceof Long) {
            var vv = ((Long) v).getUnderlying();
            var bs = BigInteger.valueOf(vv).toByteArray();
            encode(bs);
        } else if (v instanceof Character) {
            var vv = ((Character) v).getUnderlying();
            var bs = BigInteger.valueOf(vv).toByteArray();
            encode(bs);
        } else {
            throw new IllegalArgumentException("bad key type :" + v.getClass());
        }
    }

    private void encode(byte[] bs) {
        int l = bs.length;
        if (l == 1 && (bs[0] & 0xFF) < SHORT_BASE) {
            bos.write(bs[0]);
        } else if (l <= SHORT_LEN_LIMIT) {
            bos.write(SHORT_BASE + l);
            bos.write(bs, 0, l);
        } else if (l <= 0xFF) {
            bos.write(LONG_BASE + 1);
            bos.write(l);
            bos.write(bs, 0, l);
        } else if (l <= 0xFFFF) {
            bos.write(LONG_BASE + 2);
            bos.write(l >> 8);
            bos.write(l);
            bos.write(bs, 0, l);
        } else if (l <= 0xFFFFFF) {
            bos.write(LONG_BASE + 3);
            bos.write(l >> 16);
            bos.write(l >> 8);
            bos.write(l);
            bos.write(bs, 0, l);
        } else {
            bos.write(LONG_BASE + 4);
            bos.write(l >> 24);
            bos.write(l >> 16);
            bos.write(l >> 8);
            bos.write(l);
            bos.write(bs, 0, l);
        }
    }
}
