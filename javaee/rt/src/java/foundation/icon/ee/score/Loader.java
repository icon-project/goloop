package foundation.icon.ee.score;

import foundation.icon.ee.types.Address;
import i.RuntimeAssertionError;
import org.aion.avm.core.AvmConfiguration;
import org.aion.avm.core.DAppLoader;
import org.aion.avm.core.IExternalState;
import org.aion.avm.core.persistence.LoadedDApp;

import java.io.IOException;
import java.lang.ref.SoftReference;
import java.util.Collections;
import java.util.LinkedHashMap;
import java.util.Map;

public class Loader {
    private static final int MAX_ENTRY = 256;

    private Map<Address, SoftReference<LoadedDApp>> dappCache = Collections.synchronizedMap(
            new LinkedHashMap<>(
                    MAX_ENTRY + 1,
                    1.0f,
                    true
            ) {
                protected boolean removeEldestEntry(Map.Entry entry) {
                    return (size() > MAX_ENTRY);
                }
            });

    public LoadedDApp load(Address addr, IExternalState es, AvmConfiguration conf) {
        var dappSR = dappCache.get(addr);
        var dapp = (dappSR != null) ? dappSR.get() : null;
        if (dapp == null) {
            if (es != null) {
                byte[] code;
                try {
                    code = es.getTransformedCode();
                } catch (IOException e) {
                    var transformer = new Transformer(es, conf);
                    transformer.transform();
                    code = transformer.getTransformedCodeBytes();
                    es.setTransformedCode(code);
                }
                try {
                    dapp = DAppLoader.loadFromGraph(code, conf.preserveDebuggability);
                } catch (IOException e) {
                    RuntimeAssertionError.unexpected(e);
                }
                if (dapp != null) {
                    dappCache.put(addr, new SoftReference<>(dapp));
                }
            }
        }
        return dapp;
    }
}
