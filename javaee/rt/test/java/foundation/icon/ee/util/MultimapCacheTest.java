package foundation.icon.ee.util;

import org.junit.Test;
import org.junit.jupiter.api.Assertions;

import java.lang.ref.WeakReference;

public class MultimapCacheTest {
    @Test
    public void testGC() {
        var mc = MultimapCache.<String, Object>newWeakCache(10);
        var o1 = new Object();
        var o2 = new Object();
        mc.put("k1", o1);
        mc.put("k2", o2);
        systemGC();
        Assertions.assertEquals(2, mc.size());
        o1 = null;
        while (mc.size() == 2) {
            systemGC();
            mc.gc();
        }
        Assertions.assertEquals(1, mc.size());
    }

    static void sleep(int millis) {
        try {
            Thread.sleep(millis);
        } catch (InterruptedException e) {
            e.printStackTrace();
        }
    }

    static void systemGC() {
        Object obj = new Object();
        var ref = new WeakReference<>(obj);
        obj = null;
        do {
            System.gc();
            sleep(100);
        } while(ref.get() != null);
    }
}
