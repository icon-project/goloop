package foundation.icon.ee;

import foundation.icon.ee.test.GoldenTest;
import org.junit.jupiter.api.Test;
import score.Address;
import score.Context;
import score.annotation.External;
import score.annotation.Optional;

public class IntercallRefTest extends GoldenTest {
    public static class RefScoreA {
        public String sString;
        public Class<?> sClass;

        @External
        public void method(int ttl, boolean ok, @Optional Address addr) {
            Context.println("enter A.method(" + ttl + " " + ok + " " + addr + ")");
            sString = "string"+ttl;
            String lString = sString;
            sClass = String.class;
            Class<?> lClass1 = String.class;
            if (ttl>0) {
                if (addr==null) {
                    addr = Context.getCaller();
                }
                try {
                    Context.call(addr, "method", ttl-1, false, null);
                } catch (Exception e) {
                    Context.println("Exception : " + e);
                }
                Context.println("lString==sString : " + (lString==sString));
                Class<?> lClass2 = String.class;
                Context.println("lClass1==lClass2 : " + (lClass1==lClass2));
                Context.println("sClass==lClass1 : " + (sClass==lClass1));
                Context.println("sClass==lClass2 : " + (sClass==lClass2));
                try {
                    Context.call(addr, "method", ttl-1, true, null);
                } catch (Exception e) {
                    Context.println("Exception : " + e);
                }
                Context.println("lString==sString : " + (lString==sString));
                lClass2 = String.class;
                Context.println("lClass1==lClass2 : " + (lClass1==lClass2));
                var lClass3 = Integer.class;
                Context.println("lClass3==sClass : " + (lClass3==sClass));
            } else {
                sClass = Integer.class;
            }
            Context.println("leave A.method");
            if (!ok) {
                Context.revert();
            }
        }
    }

    public static class RefScoreB {
        @External
        public void method(int ttl, boolean ok, @Optional Address addr) {
            Context.println("enter B.method(" + ttl + " " + ok + " " + addr + ")");
            if (ttl>0) {
                if (addr==null) {
                    addr = Context.getCaller();
                }
                try {
                    Context.call(addr, "method", ttl-1, false, null);
                } catch (Exception e) {
                    Context.println("Exception : " + e);
                }
                try {
                    Context.call(addr, "method", ttl-1, true, null);
                } catch (Exception e) {
                    Context.println("Exception : " + e);
                }
            }
            Context.println("leave B.method");
            if (!ok) {
                Context.revert();
            }
        }
    }

    @Test
    public void testRef1() {
        var app1 = sm.mustDeploy(RefScoreA.class);
        var app2 = sm.mustDeploy(RefScoreB.class);
        app1.invoke("method", 1, true, app2.getAddress());
    }

    @Test
    public void testRef2() {
        var app1 = sm.mustDeploy(RefScoreA.class);
        var app2 = sm.mustDeploy(RefScoreB.class);
        app1.invoke("method", 2, true, app2.getAddress());
    }

    @Test
    public void testRef3() {
        createAndAcceptNewJAVAEE();
        var app1 = sm.mustDeploy(RefScoreA.class);
        sm.setIndexer((addr) -> 1);
        var app2 = sm.mustDeploy(RefScoreB.class);
        sm.setIndexer((addr) -> {
            if (addr.equals(app1.getAddress())) {
                return 0;
            }
            return 1;
        });
        app1.invoke("method", 2, true, app2.getAddress());
    }
}
