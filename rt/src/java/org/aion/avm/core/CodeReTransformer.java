package org.aion.avm.core;

import i.AvmException;
import i.JvmError;
import org.aion.avm.core.rejection.RejectedClassException;
import org.aion.avm.core.types.ImmortalDappModule;
import org.aion.avm.core.types.RawDappModule;

import java.util.Map;

/**
 * This class performs the class re-transformation similar to the DAppCreator, except the classes are not initialized and certain error cases should not happen.
 * This includes incorrectly packaged jar, corrupted jar, missing Main class, missing main() method, OutOfEnergyException, CallDepthLimitExceededException,
 * RevertException, InvalidException, UncaughtException, OutOfStackException, UncaughtException, EarlyAbortException.
 */
public class CodeReTransformer {

    public static byte[] transformCode(byte[] code, long blockTimeStamp, boolean preserveDebuggability, boolean verboseErrors) {
        byte[] transformedCode = null;
        try {
            RawDappModule rawDapp = RawDappModule.readFromJar(code, preserveDebuggability, verboseErrors);
            Map<String, byte[]> transformedClasses = DAppCreator.transformClasses(rawDapp.classes, rawDapp.classHierarchyForest, rawDapp.classHierarchy, rawDapp.classRenamer, preserveDebuggability);
            Map<String, byte[]> immortalClasses = DAppCreator.stripClinitFromClasses(transformedClasses);
            ImmortalDappModule immortalDapp = ImmortalDappModule.fromImmortalClasses(immortalClasses, rawDapp.mainClass);
            transformedCode = immortalDapp.createJar(blockTimeStamp);

        } catch (RejectedClassException e) {
            if (verboseErrors) {
                System.err.println("DApp re-transformation REJECTED with reason: \"" + e.getMessage() + "\"");
                e.printStackTrace(System.err);
            }
        } catch (AvmException e) {
            if (verboseErrors) {
                System.err.println("DApp re-transformation failed due to AvmException: \"" + e.getMessage() + "\"");
                e.printStackTrace(System.err);
            }
        } catch (JvmError e) {
            if (verboseErrors) {
                System.err.println("FATAL JvmError: \"" + e.getMessage() + "\"");
                e.printStackTrace(System.err);
            }
            throw e;
        } catch (Throwable e) {
            if (verboseErrors) {
                System.err.println("Unknown error when re-transformation this code: \"" + e.getMessage() + "\"");
                e.printStackTrace(System.err);
            }
        }
        return transformedCode;
    }
}
