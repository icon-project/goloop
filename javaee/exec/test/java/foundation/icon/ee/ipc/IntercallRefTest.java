package foundation.icon.ee.ipc;

import avm.Address;
import avm.Blockchain;
import foundation.icon.ee.test.GoldenTest;
import foundation.icon.ee.tooling.abi.External;
import foundation.icon.ee.tooling.abi.Optional;
import org.junit.jupiter.api.Test;

import java.math.BigInteger;

public class IntercallRefTest extends GoldenTest {
    public static class RefScoreA {
        public static String sString;
        public static Class<?> sClass;

        @External
        public static void method(int ttl, boolean ok, @Optional Address addr) {
            Blockchain.println("enter A.method(" + ttl + " " + ok + " " + addr + ")");
            sString = "string"+ttl;
            String lString = sString;
            sClass = String.class;
            Class<?> lClass1 = String.class;
            if (ttl>0) {
                if (addr==null) {
                    addr = Blockchain.getCaller();
                }
                try {
                    Blockchain.call(addr, "method", ttl-1, false, null);
                } catch (Exception e) {
                    Blockchain.println("Exception : " + e);
                }
                Blockchain.println("lString==sString : " + (lString==sString));
                Class<?> lClass2 = String.class;
                Blockchain.println("lClass1==lClass2 : " + (lClass1==lClass2));
                Blockchain.println("sClass==lClass1 : " + (sClass==lClass1));
                Blockchain.println("sClass==lClass2 : " + (sClass==lClass2));
                try {
                    Blockchain.call(addr, "method", ttl-1, true, null);
                } catch (Exception e) {
                    Blockchain.println("Exception : " + e);
                }
                Blockchain.println("lString==sString : " + (lString==sString));
                lClass2 = String.class;
                Blockchain.println("lClass1==lClass2 : " + (lClass1==lClass2));
                var lClass3 = Integer.class;
                Blockchain.println("lClass3==sClass : " + (lClass3==sClass));
            } else {
                sClass = Integer.class;
            }
            Blockchain.println("leave A.method");
            if (!ok) {
                Blockchain.revert();
            }
        }
    }

    public static class RefScoreB {
        @External
        public static void method(int ttl, boolean ok, @Optional Address addr) {
            Blockchain.println("enter B.method(" + ttl + " " + ok + " " + addr + ")");
            if (ttl>0) {
                if (addr==null) {
                    addr = Blockchain.getCaller();
                }
                try {
                    Blockchain.call(addr, "method", ttl-1, false, null);
                } catch (Exception e) {
                    Blockchain.println("Exception : " + e);
                }
                try {
                    Blockchain.call(addr, "method", ttl-1, true, null);
                } catch (Exception e) {
                    Blockchain.println("Exception : " + e);
                }
            }
            Blockchain.println("leave B.method");
            if (!ok) {
                Blockchain.revert();
            }
        }
    }

    @Test
    public void testRef1() {
        var app1 = sm.deploy(RefScoreA.class);
        var app2 = sm.deploy(RefScoreB.class);
        app1.invoke("method", 1, true, app2.getAddress());
    }

    @Test
    public void testRef2() {
        var app1 = sm.deploy(RefScoreA.class);
        var app2 = sm.deploy(RefScoreB.class);
        app1.invoke("method", 2, true, app2.getAddress());
    }
}
