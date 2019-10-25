package org.aion.avm.core;

import i.AvmException;
import i.JvmError;
import org.aion.avm.core.rejection.RejectedClassException;
import org.aion.avm.core.types.ImmortalDappModule;
import org.aion.avm.core.types.RawDappModule;

import java.util.List;
import java.util.Map;
import java.util.stream.Collectors;

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

            if (canRetransform(rawDapp)) {
                Map<String, byte[]> transformedClasses = DAppCreator.transformClasses(rawDapp.classes, rawDapp.classHierarchyForest, rawDapp.classHierarchy, rawDapp.classRenamer, preserveDebuggability);
                Map<String, byte[]> immortalClasses = DAppCreator.stripClinitFromClasses(transformedClasses);
                ImmortalDappModule immortalDapp = ImmortalDappModule.fromImmortalClasses(immortalClasses, rawDapp.mainClass);
                transformedCode = immortalDapp.createJar(blockTimeStamp);
            }

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

    // AKI-329: if a DApp has been deployed which contains an interface with an inner class/interface called FIELDS, transaction will fail.
    // This is because object graph will no longer match the transformed code and deserialization will fail.
    // In this case DApp will not be re-transformed and transaction will fail with FAILED_RETRANSFORMATION error
    private static boolean canRetransform(RawDappModule rawDapp) {
        boolean canDAppBeRetransformed = true;
        List<String> classNamesContainingFIELDS = rawDapp.classes.keySet()
                .stream()
                .filter(n -> n.endsWith("$FIELDS"))
                .map(n -> rawDapp.classRenamer.toPostRename(n, ClassRenamer.ArrayType.NOT_ARRAY))
                .collect(Collectors.toList());
        for (String name : classNamesContainingFIELDS) {
            String outerClass = name.substring(0, name.length() - 7);
            if (rawDapp.classHierarchy.postRenameTypeIsInterface(outerClass)) {
                canDAppBeRetransformed = false;
                break;
            }
        }
        return canDAppBeRetransformed;
    }
}
