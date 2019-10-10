/*
 * Copyright 2019 ICON Foundation
 * Copyright (c) 2018 Aion Foundation https://aion.network/
 */

package foundation.icon.ee.tooling.deploy;

import foundation.icon.ee.tooling.abi.ABICompiler;
import foundation.icon.ee.types.Method;
import foundation.icon.ee.utils.MethodPacker;
import org.aion.avm.tooling.deploy.JarOptimizer;
import org.aion.avm.tooling.deploy.eliminator.ConstantRemover;
import org.aion.avm.tooling.deploy.eliminator.UnreachableMethodRemover;
import org.aion.avm.tooling.deploy.renamer.Renamer;
import org.aion.avm.tooling.util.JarBuilder;
import org.aion.avm.tooling.util.Utilities;
import org.msgpack.core.MessageBufferPacker;
import org.msgpack.core.MessagePack;

import java.io.ByteArrayInputStream;
import java.io.IOException;
import java.util.List;
import java.util.Map;
import java.util.jar.JarInputStream;

public class OptimizedJarBuilder {

    private boolean debugModeEnabled;
    private boolean unreachableMethodRemoverEnabled;
    private boolean classAndFieldRenamerEnabled;
    private boolean constantRemoverEnabled;
    private byte[] dappBytes;
    private List<Method> callables;

    /**
     * Initializes a new instance of OptimizedJarBuilder, which allows desired optimization steps to be enabled and performed
     * @param debugModeEnabled Indicates if debug data and names need to be preserved
     * @param jarBytes Byte array corresponding to the jar
     * @param abiVersion Version of ABI compiler to use
     */
    public OptimizedJarBuilder(boolean debugModeEnabled, byte[] jarBytes, int abiVersion) {
        this.debugModeEnabled = debugModeEnabled;
        ABICompiler compiler = ABICompiler.compileJarBytes(jarBytes, abiVersion);
        dappBytes = compiler.getJarFileBytes();
        callables = compiler.getCallables();
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

    /**
     * Removes ABIException messages from the contract to reduce number of string constants
     * @return OptimizedJarBuilder
     */
    public OptimizedJarBuilder withConstantRemover() {
        constantRemoverEnabled = true;
        return this;
    }

    /**
     * Performs selected optimization steps.
     * Unreferenced classes are removed from the Jar for all cases.
     * @return optimized jar bytes
     */
    public byte[] getOptimizedBytes() {
        JarOptimizer jarOptimizer = new JarOptimizer(debugModeEnabled);
        byte[] optimizedDappBytes = jarOptimizer.optimize(dappBytes);
        if (constantRemoverEnabled) {
            try {
                optimizedDappBytes = ConstantRemover.removeABIExceptionMessages(optimizedDappBytes);
            } catch (Exception exception) {
                System.err.println("Constant Remover crashed, packaging code without this optimization");
            }
        }
        if (unreachableMethodRemoverEnabled) {
            try {
                optimizedDappBytes = UnreachableMethodRemover.optimize(optimizedDappBytes);

                // Run class removal optimization again to ensure classes without any referenced methods are removed
                optimizedDappBytes = jarOptimizer.optimize(optimizedDappBytes);
            } catch (Exception exception) {
                System.err.println("UnreachableMethodRemover crashed, packaging code without this optimization");
            }
        }
        // Renaming is disabled in debug mode.
        // Only field and method renaming can work correctly in debug mode, but the new names may cause confusion for users.
        if (classAndFieldRenamerEnabled && !debugModeEnabled) {
            try {
                optimizedDappBytes = Renamer.rename(optimizedDappBytes);
            } catch (Exception exception) {
                System.err.println("Renaming crashed, packaging code without this optimization");
            }
        }
        // Add API info into the Jar
        try {
            optimizedDappBytes = writeApi(optimizedDappBytes);
        } catch (Exception e) {
            System.err.println("Writing API info failed.");
        }
        return optimizedDappBytes;
    }

    public List<Method> getCallables() {
        return callables;
    }

    private byte[] writeApi(byte[] jarBytes) throws IOException {
        MessageBufferPacker packer = MessagePack.newDefaultBufferPacker();
        packer.packArrayHeader(callables.size());
        for (Method m : callables) {
            if (debugModeEnabled) {
                System.out.println(m);
            }
            MethodPacker.writeTo(m, packer);
        }
        packer.close();

        JarInputStream jis = new JarInputStream(new ByteArrayInputStream(jarBytes), true);
        Map<String, byte[]> classMap = Utilities.extractClasses(jis, Utilities.NameStyle.DOT_NAME);
        String mainClassName = Utilities.extractMainClassName(jis, Utilities.NameStyle.DOT_NAME);
        byte[] mainClassBytes = classMap.remove(mainClassName);
        return JarBuilder.buildJarWithApiInfo(mainClassName, mainClassBytes, packer.toByteArray(), classMap);
    }
}
