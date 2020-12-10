package foundation.icon.ee.util;

import java.lang.ref.ReferenceQueue;
import java.util.concurrent.ScheduledExecutorService;
import java.util.concurrent.ScheduledThreadPoolExecutor;
import java.util.concurrent.TimeUnit;

public class DisposableReferenceQueue<T> extends ReferenceQueue<T> {
    private static final int DEFAULT_CONSUME_INTERVAL = 300;
    private static class Lazy {
        private static final ScheduledExecutorService ses = newSES();

        private static ScheduledExecutorService newSES() {
            var stpe = new ScheduledThreadPoolExecutor(1);
            stpe.allowCoreThreadTimeOut(true);
            return stpe;
        }
    }

    public DisposableReferenceQueue() {
        this(null, 0);
    }

    private DisposableReferenceQueue(ScheduledExecutorService ses,
            int intervalMilli) {
        if (ses == null) {
            ses = Lazy.ses;
        }
        if (intervalMilli <= 0) {
            intervalMilli = DEFAULT_CONSUME_INTERVAL;
        }
        ses.scheduleAtFixedRate(this::consumeAll,
                intervalMilli,
                intervalMilli,
                TimeUnit.MILLISECONDS);
    }

    public void consumeAll() {
        Disposable d;
        while ((d=(Disposable) poll()) != null) {
            d.dispose();
        }
    }
}
