package foundation.icon.ee.score;

import org.aion.avm.core.DAppLoader;
import org.aion.avm.core.IExternalState;
import org.aion.avm.core.persistence.LoadedDApp;
import org.aion.types.AionAddress;

import java.io.IOException;
import java.lang.ref.SoftReference;
import java.util.LinkedHashMap;
import java.util.Map;

public class Loader {
    private static final int MAX_ENTRY = 256;

    private LinkedHashMap<AionAddress, SoftReference<LoadedDApp>> dappCache= new LinkedHashMap<>(
            MAX_ENTRY+1,
            1.0f,
            true
    ) {
        protected boolean removeEldestEntry(Map.Entry entry) {
            return (size() > MAX_ENTRY);
        }
    };

    public LoadedDApp load(AionAddress addr, IExternalState es, boolean preserveDebuggability) throws IOException {
        synchronized(this) {
            var dappSR = dappCache.get(addr);
            var dapp = (dappSR!=null) ? dappSR.get() : null;
            if (dapp == null) {
                if (es != null) {
                    var code = es.getTransformedCode(addr);
                    dapp = DAppLoader.loadFromGraph(code, preserveDebuggability);
                    if (dapp != null) {
                        dappCache.put(addr, new SoftReference<>(dapp));
                    }
                }
            }
            return dapp;
        }
    }
}
