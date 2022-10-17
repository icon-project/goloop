/*
 * Copyright 2019 ICON Foundation
 * Copyright (c) 2018 Aion Foundation https://aion.network/
 */

package foundation.icon.ee.tooling.deploy;

import foundation.icon.ee.struct.Member;
import foundation.icon.ee.tooling.abi.ABICompiler;
import foundation.icon.ee.types.Method;
import foundation.icon.ee.util.MethodPacker;
import org.aion.avm.tooling.deploy.JarOptimizer;
import org.aion.avm.tooling.deploy.eliminator.UnreachableMethodRemover;
import org.aion.avm.tooling.deploy.renamer.Renamer;
import org.aion.avm.utilities.JarBuilder;
import org.aion.avm.utilities.Utilities;
import org.msgpack.core.MessageBufferPacker;
import org.msgpack.core.MessagePack;

import java.io.ByteArrayInputStream;
import java.io.IOException;
import java.io.PrintStream;
import java.util.List;
import java.util.Map;
import java.util.Set;
import java.util.jar.JarInputStream;

public class OptimizedJarBuilder {

    private final boolean debugModeEnabled;
    private boolean unreachableMethodRemoverEnabled;
    private boolean classAndFieldRenamerEnabled;
    private PrintStream log = null;
    private final byte[] dappBytes;
    private List<Method> callables;
    private final Set<String> rootClasses;
    private final Map<String, List<Member>> keptMethods;
    private final Map<String, List<Member>> keptFields;

    /**
     * Initializes a new instance of OptimizedJarBuilder, which allows desired optimization steps to be enabled and performed
     * @param debugModeEnabled Indicates if debug data and names need to be preserved
     * @param jarBytes Byte array corresponding to the jar
     */
    public OptimizedJarBuilder(boolean debugModeEnabled, byte[] jarBytes) {
        this(debugModeEnabled, jarBytes, false);
    }

    public OptimizedJarBuilder(boolean debugModeEnabled, byte[] jarBytes, boolean stripLineNumber) {
        this.debugModeEnabled = debugModeEnabled;
        ABICompiler compiler = ABICompiler.compileJarBytes(jarBytes, stripLineNumber);
        dappBytes = compiler.getJarFileBytes();
        callables = compiler.getCallables();
        rootClasses = compiler.getRootClasses();
        keptMethods = compiler.getKeptMethods();
        keptFields = compiler.getKeptFields();
    }

    /**
     * Removes methods not reachable from main method
     * @return OptimizedJarBuilder
     */
    public OptimizedJarBuilder withUnreachableMethodRemover() {
        unreachableMethodRemoverEnabled = true;
        return this;
    }

    /**
     * Renames all the class, method, and field names to smaller names (starting from character names)
     * @return OptimizedJarBuilder
     */
    public OptimizedJarBuilder withRenamer() {
        classAndFieldRenamerEnabled = true;
        return this;
    }

    public OptimizedJarBuilder withLog(PrintStream log) {
        this.log = log;
        return this;
    }

    /**
     * Performs selected optimization steps.
     * Unreferenced classes are removed from the Jar for all cases.
     * @return optimized jar bytes
     */
    public byte[] getOptimizedBytes() {
        JarOptimizer jarOptimizer = new JarOptimizer(debugModeEnabled);
        byte[] optimizedDappBytes = jarOptimizer.optimize(dappBytes,
                rootClasses);
        if (unreachableMethodRemoverEnabled) {
            try {
                optimizedDappBytes = UnreachableMethodRemover.optimize(optimizedDappBytes, keptMethods);

                // Run class removal optimization again to ensure classes without any referenced methods are removed
                optimizedDappBytes = jarOptimizer.optimize(optimizedDappBytes,
                        rootClasses);
            } catch (UnsupportedOperationException ex) {
                throw ex;
            } catch (Exception exception) {
                System.err.println("UnreachableMethodRemover failed, packaging code without this optimization");
                exception.printStackTrace(System.err);
            }
        }
        // Renaming is disabled in debug mode.
        // Only field and method renaming can work correctly in debug mode, but the new names may cause confusion for users.
        if (classAndFieldRenamerEnabled && !debugModeEnabled) {
            try {
                var res = Renamer.rename(optimizedDappBytes,
                        callables, keptMethods, keptFields, log);
                optimizedDappBytes = res.getJarBytes();
                callables = res.getCallables();
            } catch (Exception exception) {
                System.err.println("Renaming failed, packaging code without this optimization");
                exception.printStackTrace(System.err);
            }
        }
        // Add API info into the Jar
        try {
            optimizedDappBytes = writeApi(optimizedDappBytes);
        } catch (Exception e) {
            System.err.println("Writing API info failed.");
            e.printStackTrace(System.err);
        }
        return optimizedDappBytes;
    }

    private byte[] writeApi(byte[] jarBytes) throws IOException {
        MessageBufferPacker packer = MessagePack.newDefaultBufferPacker();
        try (packer) {
            packer.packArrayHeader(callables.size());
            for (Method m : callables) {
                if (debugModeEnabled) {
                    System.out.println(m);
                }
                MethodPacker.writeTo(m, packer, true);
            }
        }

        JarInputStream jis = new JarInputStream(new ByteArrayInputStream(jarBytes), true);
        Map<String, byte[]> classMap = Utilities.extractClasses(jis, Utilities.NameStyle.DOT_NAME);
        String mainClassName = Utilities.extractMainClassName(jis, Utilities.NameStyle.DOT_NAME);
        byte[] mainClassBytes = classMap.remove(mainClassName);
        return JarBuilder.buildJarWithApiInfo(mainClassName, mainClassBytes, packer.toByteArray(), classMap);
    }
}
