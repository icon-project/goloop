package foundation.icon.ee.score;

import foundation.icon.ee.util.MultimapCache;
import i.RuntimeAssertionError;
import org.aion.avm.core.AvmConfiguration;
import org.aion.avm.core.DAppLoader;
import org.aion.avm.core.IExternalState;
import org.aion.avm.core.persistence.LoadedDApp;

import java.io.IOException;

public class Loader {
    private static final int CAP = 256;

    private final MultimapCache<String, LoadedDApp> cache =
            MultimapCache.newSoftCache(CAP);

    public LoadedDApp load(IExternalState es, AvmConfiguration conf) {
        var dapp = cache.remove(es.getCodeID(), da ->
                da.hasSameGraphHash(es.getObjectGraphHash())
        );
        if (dapp != null) {
            if (es.purgeEnumCache()) {
                dapp.getInternedClasses().purgeEnumCaches();
            }
        } else {
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
                dapp = DAppLoader.loadFromGraph(code,
                        conf.preserveDebuggability);
            } catch (IOException e) {
                RuntimeAssertionError.unexpected(e);
            }
        }
        return dapp;
    }

    public void unload(String codeID, LoadedDApp dapp) {
        cache.put(codeID, dapp);
    }
}
