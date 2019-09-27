package org.aion.avm.core;

import org.aion.avm.core.util.TransactionResultUtil;
import org.aion.kernel.AvmWrappedTransactionResult.AvmInternalError;
import org.aion.types.AionAddress;
import org.aion.types.Transaction;
import org.aion.avm.RuntimeMethodFeeSchedule;
import org.aion.avm.StorageFees;
import org.aion.avm.core.ClassRenamer.ArrayType;
import org.aion.avm.core.arraywrapping.ArrayWrappingClassAdapter;
import org.aion.avm.core.arraywrapping.ArrayWrappingClassAdapterRef;
import org.aion.avm.core.exceptionwrapping.ExceptionWrapping;
import org.aion.avm.core.instrument.ClassMetering;
import org.aion.avm.core.instrument.HeapMemoryCostCalculator;
import org.aion.avm.core.miscvisitors.ClinitStrippingVisitor;
import org.aion.avm.core.miscvisitors.ConstantVisitor;
import org.aion.avm.core.miscvisitors.InterfaceFieldMappingVisitor;
import org.aion.avm.core.miscvisitors.LoopingExceptionStrippingVisitor;
import org.aion.avm.core.miscvisitors.NamespaceMapper;
import org.aion.avm.core.miscvisitors.PreRenameClassAccessRules;
import org.aion.avm.core.miscvisitors.StrictFPVisitor;
import org.aion.avm.core.miscvisitors.UserClassMappingVisitor;
import org.aion.avm.core.persistence.AutomaticGraphVisitor;
import org.aion.avm.core.persistence.LoadedDApp;
import org.aion.avm.core.rejection.InstanceVariableCountManager;
import org.aion.avm.core.rejection.InstanceVariableCountingVisitor;
import org.aion.avm.core.rejection.MainMethodChecker;
import org.aion.avm.core.rejection.RejectedClassException;
import org.aion.avm.core.rejection.RejectionClassVisitor;
import org.aion.avm.core.shadowing.ClassShadowing;
import org.aion.avm.core.shadowing.InvokedynamicShadower;
import org.aion.avm.core.stacktracking.StackWatcherClassAdapter;
import org.aion.avm.core.types.ClassHierarchy;
import org.aion.avm.core.types.ClassInfo;
import org.aion.avm.core.types.Forest;
import org.aion.avm.core.types.GeneratedClassConsumer;
import org.aion.avm.core.types.ImmortalDappModule;
import org.aion.avm.core.types.RawDappModule;
import org.aion.avm.core.types.TransformedDappModule;
import org.aion.avm.core.util.DebugNameResolver;
import org.aion.avm.core.util.Helpers;
import org.aion.avm.core.verification.Verifier;
import org.aion.avm.userlib.CodeAndArguments;

import i.*;
import org.aion.kernel.*;
import org.aion.parallel.TransactionTask;
import org.objectweb.asm.ClassReader;
import org.objectweb.asm.ClassWriter;

import java.util.*;


public class DAppCreator {
    /**
     * Returns the sizes of all the user-space classes
     *
     * @param classHierarchy     the class hierarchy
     * @return The look-up map of the sizes of user objects
     * Class name is in the JVM internal name format, see {@link org.aion.avm.core.util.Helpers#fulllyQualifiedNameToInternalName(String)}
     */
    public static Map<String, Integer> computeUserObjectSizes(Forest<String, ClassInfo> classHierarchy, Map<String, Integer> rootObjectSizes)
    {
        HeapMemoryCostCalculator objectSizeCalculator = new HeapMemoryCostCalculator();

        // compute the user object sizes
        objectSizeCalculator.calcClassesInstanceSize(classHierarchy, rootObjectSizes);

        // copy over the user object sizes
        Map<String, Integer> userObjectSizes = new HashMap<>();
        objectSizeCalculator.getClassHeapSizeMap().forEach((k, v) -> {
            if (!rootObjectSizes.containsKey(k)) {
                userObjectSizes.put(k, v);
            }
        });
        return userObjectSizes;
    }

    private static Map<String, Integer> computeAllPostRenameObjectSizes(Forest<String, ClassInfo> forest, boolean preserveDebuggability) {
        Map<String, Integer> preRenameUserObjectSizes = computeUserObjectSizes(forest, NodeEnvironment.singleton.preRenameRuntimeObjectSizeMap);

        Map<String, Integer> postRenameObjectSizes = new HashMap<>(NodeEnvironment.singleton.postRenameRuntimeObjectSizeMap);
        preRenameUserObjectSizes.forEach((k, v) -> postRenameObjectSizes.put(DebugNameResolver.getUserPackageSlashPrefix(k, preserveDebuggability), v));
        return postRenameObjectSizes;
    }

    /**
     * Replaces the <code>java.base</code> package with the shadow implementation.
     * Note that this is public since some unit tests call it, directly.
     *
     * @param inputClasses The class of DApp (names specified in .-style)
     * @param oldPreRenameForest The pre-rename forest of user-defined classes in the DApp (/-style).
     * @param classHierarchy The class hierarchy of all classes in the system (.-style).
     * @param preserveDebuggability Whether or not debug mode is enabled.
     * @return the transformed classes and any generated classes (names specified in .-style)
     */
    public static Map<String, byte[]> transformClasses(Map<String, byte[]> inputClasses, Forest<String, ClassInfo> oldPreRenameForest, ClassHierarchy classHierarchy, ClassRenamer classRenamer, boolean preserveDebuggability) {
        // Before anything, pass the list of classes through the verifier.
        // (this will throw UncaughtException, on verification failure).
        Verifier.verifyUntrustedClasses(inputClasses);
        // We need to run our rejection filter and static rename pass.
        Map<String, byte[]> safeClasses = rejectionAndRenameInputClasses(inputClasses, classHierarchy, classRenamer, preserveDebuggability);
        
        ConstantClassBuilder.ConstantClassInfo constantClass = ConstantClassBuilder.buildConstantClassBytecodeForClasses(PackageConstants.kConstantClassName, safeClasses.values());
        
        // merge the generated classes and processed classes, assuming the package spaces do not conflict.
        Map<String, byte[]> processedClasses = new HashMap<>();
        
        // Start by adding the constant class.
        processedClasses.put(PackageConstants.kConstantClassName, constantClass.bytecode);
        
        // merge the generated classes and processed classes, assuming the package spaces do not conflict.
        // We also want to expose this type to the class writer so it can compute common superclasses.
        GeneratedClassConsumer generatedClassesSink = (superClassSlashName, classSlashName, bytecode) -> {
            // Note that the processed classes are expected to use .-style names.
            String classDotName = Helpers.internalNameToFulllyQualifiedName(classSlashName);
            processedClasses.put(classDotName, bytecode);
        };
        Map<String, Integer> postRenameObjectSizes = computeAllPostRenameObjectSizes(oldPreRenameForest, preserveDebuggability);

        Map<String, byte[]> transformedClasses = new HashMap<>();

        int parsingOptions = preserveDebuggability ? ClassReader.EXPAND_FRAMES : ClassReader.EXPAND_FRAMES | ClassReader.SKIP_DEBUG;

        for (String name : safeClasses.keySet()) {
            // Note that transformClasses requires that the input class names by the .-style names.
            RuntimeAssertionError.assertTrue(-1 == name.indexOf("/"));

            // We need to parse with EXPAND_FRAMES, since the StackWatcherClassAdapter uses a MethodNode to parse methods.
            // We also add SKIP_DEBUG since we aren't using debug data and skipping it removes extraneous labels which would otherwise
            // cause the BlockBuildingMethodVisitor to build lots of small blocks instead of a few big ones (each block incurs a Helper
            // static call, which is somewhat expensive - this is how we bill for energy).
            byte[] bytecode = new ClassToolchain.Builder(safeClasses.get(name), parsingOptions)
                    .addNextVisitor(new ClassMetering(postRenameObjectSizes))
                    .addNextVisitor(new ConstantVisitor(PackageConstants.kConstantClassName, constantClass.constantToFieldMap))
                    .addNextVisitor(new InvokedynamicShadower(PackageConstants.kShadowSlashPrefix))
                    .addNextVisitor(new ClassShadowing(PackageConstants.kShadowSlashPrefix))
                    .addNextVisitor(new StackWatcherClassAdapter())
                    .addNextVisitor(new ExceptionWrapping(generatedClassesSink, classHierarchy))
                    .addNextVisitor(new AutomaticGraphVisitor())
                    .addNextVisitor(new StrictFPVisitor())
                    .addWriter(new TypeAwareClassWriter(ClassWriter.COMPUTE_FRAMES | ClassWriter.COMPUTE_MAXS, classHierarchy, classRenamer))
                    .build()
                    .runAndGetBytecode();
            bytecode = new ClassToolchain.Builder(bytecode, parsingOptions)
                    .addNextVisitor(new ArrayWrappingClassAdapterRef(classHierarchy))
                    .addNextVisitor(new ArrayWrappingClassAdapter())
                    .addWriter(new TypeAwareClassWriter(ClassWriter.COMPUTE_FRAMES | ClassWriter.COMPUTE_MAXS, classHierarchy, classRenamer))
                    .build()
                    .runAndGetBytecode();
            transformedClasses.put(name, bytecode);
        }

        /*
         * Another pass to deal with static fields in interfaces.
         * Note that all fields in interfaces are defined as static.
         */

        // We grab all of the user-defined interfaces.
        Set<String> preRenameUserClassesAndInterfaces = classHierarchy.getPreRenameUserDefinedClassesAndInterfaces();
        Set<String> userInterfaceSlashNames = new HashSet<>();

        for (String preRenameUserClassOrInterface : preRenameUserClassesAndInterfaces) {
            // We set ArrayType to null because we never expect to see arrays here!
            String classNamePostRename = classRenamer.toPostRename(preRenameUserClassOrInterface, ArrayType.NOT_ARRAY);
            if (classHierarchy.postRenameTypeIsInterface(classNamePostRename)) {
                userInterfaceSlashNames.add(Helpers.fulllyQualifiedNameToInternalName(classNamePostRename));
            }
        }

        String javaLangObjectSlashName = PackageConstants.kShadowSlashPrefix + "java/lang/Object";
        for (String name : transformedClasses.keySet()) {
            byte[] bytecode = new ClassToolchain.Builder(transformedClasses.get(name), parsingOptions)
                    .addNextVisitor(new InterfaceFieldMappingVisitor(generatedClassesSink, userInterfaceSlashNames, javaLangObjectSlashName))
                    .addWriter(new TypeAwareClassWriter(ClassWriter.COMPUTE_FRAMES | ClassWriter.COMPUTE_MAXS, classHierarchy, classRenamer))
                    .build()
                    .runAndGetBytecode();

            processedClasses.put(name, bytecode);
        }

        return processedClasses;
    }

    public static AvmWrappedTransactionResult create(IExternalCapabilities capabilities, IExternalState externalState, AvmInternal avm, TransactionTask task, Transaction tx, AvmWrappedTransactionResult internalResult, boolean preserveDebuggability, boolean verboseErrors, boolean enableBlockchainPrintln) {
        // Expose the DApp outside the try so we can detach from it, when we exit.
        LoadedDApp dapp = null;
        AvmWrappedTransactionResult result = internalResult;
        try {
            // read dapp module
            AionAddress dappAddress = (tx.isCreate) ? capabilities.generateContractAddress(tx) : tx.destinationAddress;
            CodeAndArguments codeAndArguments = CodeAndArguments.decodeFromBytes(tx.copyOfTransactionData());
            if (codeAndArguments == null) {
                if (verboseErrors) {
                    System.err.println("DApp deployment failed due to incorrectly packaged JAR and initialization arguments");
                }
                return TransactionResultUtil.newResultWithNonRevertedFailureAndEnergyUsed(AvmInternalError.FAILED_INVALID_DATA, tx.energyLimit);
            }

            RawDappModule rawDapp = RawDappModule.readFromJar(codeAndArguments.code, preserveDebuggability, verboseErrors);
            if (rawDapp == null) {
                if (verboseErrors) {
                    System.err.println("DApp deployment failed due to corrupt JAR data");
                }
                return TransactionResultUtil.newResultWithNonRevertedFailureAndEnergyUsed(AvmInternalError.FAILED_INVALID_DATA, tx.energyLimit);
            }

            // Verify that the DApp contains the main class they listed and that it has a "public static byte[] main()" method.
            if (!rawDapp.classes.containsKey(rawDapp.mainClass) || !MainMethodChecker.checkForMain(rawDapp.classes.get(rawDapp.mainClass))) {
                if (verboseErrors) {
                    String explanation = !rawDapp.classes.containsKey(rawDapp.mainClass) ? "missing Main class" : "missing main() method";
                    System.err.println("DApp deployment failed due to " + explanation);
                }
                return TransactionResultUtil.newResultWithNonRevertedFailureAndEnergyUsed(AvmInternalError.FAILED_INVALID_DATA, tx.energyLimit);
            }
            ClassHierarchyForest dappClassesForest = rawDapp.classHierarchyForest;

            // transform
            Map<String, byte[]> transformedClasses = transformClasses(rawDapp.classes, dappClassesForest, rawDapp.classHierarchy, rawDapp.classRenamer, preserveDebuggability);
            TransformedDappModule transformedDapp = TransformedDappModule.fromTransformedClasses(transformedClasses, rawDapp.mainClass);

            dapp = DAppLoader.fromTransformed(transformedDapp, preserveDebuggability);
            
            // We start the nextHashCode at 1.
            int nextHashCode = 1;
            InstrumentationHelpers.pushNewStackFrame(dapp.runtimeSetup, dapp.loader, tx.energyLimit - result.energyUsed(), nextHashCode, new InternedClasses());
            // (we pass a null reentrant state since we haven't finished initializing yet - nobody can call into us).
            IBlockchainRuntime previousRuntime = dapp.attachBlockchainRuntime(new BlockchainRuntimeImpl(capabilities, externalState, avm, null, task, tx, codeAndArguments.arguments, dapp.runtimeSetup, enableBlockchainPrintln));

            // We have just created this dApp, there should be no previous runtime associated with it.
            RuntimeAssertionError.assertTrue(previousRuntime == null);

            IInstrumentation threadInstrumentation = IInstrumentation.attachedThreadInstrumentation.get();
            threadInstrumentation.chargeEnergy(BillingRules.getDeploymentFee(rawDapp.numberOfClasses, rawDapp.bytecodeSize));

            // Create the immortal version of the transformed DApp code by stripping the <clinit>.
            Map<String, byte[]> immortalClasses = stripClinitFromClasses(transformedClasses);

            ImmortalDappModule immortalDapp = ImmortalDappModule.fromImmortalClasses(immortalClasses, transformedDapp.mainClass);

            // store deployed code
            externalState.putCode(dappAddress, codeAndArguments.code);
            // store transformed dapp
            byte[] immortalDappJar = immortalDapp.createJar(externalState.getBlockTimestamp());
            externalState.setTransformedCode(dappAddress, immortalDappJar);

            // Force the classes in the dapp to initialize so that the <clinit> is run (since we already saved the version without).
            dapp.forceInitializeAllClasses();

            // Save back the state before we return.
            byte[] rawGraphData = dapp.saveEntireGraph(threadInstrumentation.peekNextHashCode(), StorageFees.MAX_GRAPH_SIZE);
            // Bill for writing this size.
            threadInstrumentation.chargeEnergy(StorageFees.WRITE_PRICE_PER_BYTE * rawGraphData.length);
            externalState.putObjectGraph(dappAddress, rawGraphData);

            long refund = 0;
            long energyUsed = tx.energyLimit - threadInstrumentation.energyLeft();
            if (task.getTransactionStackDepth() == 0) {
                // refund is calculated for the transaction if it included a selfdestruct operation or it set the storage value from nonzero to zero
                long selfDestructRefund = 0l;
                long resetStorageRefund = 0l;

                if (task.getSelfDestructAddressCount() > 0) {
                    selfDestructRefund = task.getSelfDestructAddressCount() * RuntimeMethodFeeSchedule.BlockchainRuntime_avm_selfDestruct_refund;
                }
                if (task.getResetStorageKeyCount() > 0) {
                    resetStorageRefund = task.getResetStorageKeyCount() * RuntimeMethodFeeSchedule.BlockchainRuntime_avm_deleteStorage_refund;
                }
                // refund is capped at half the energy used for the whole transaction
                refund = Math.min(energyUsed / 2, selfDestructRefund + resetStorageRefund);
            }

            // Return data of a CREATE transaction is the new DApp address.
            result = TransactionResultUtil.newSuccessfulResultWithEnergyUsedAndOutput(energyUsed - refund, dappAddress.toByteArray());
        } catch (OutOfEnergyException e) {
            if (verboseErrors) {
                System.err.println("DApp deployment failed due to Out-of-Energy EXCEPTION: \"" + e.getMessage() + "\"");
                e.printStackTrace(System.err);
            }
            result = TransactionResultUtil.setNonRevertedFailureAndEnergyUsed(result, AvmInternalError.FAILED_OUT_OF_ENERGY, tx.energyLimit);

        } catch (OutOfStackException e) {
            if (verboseErrors) {
                System.err.println("DApp deployment failed due to stack overflow EXCEPTION: \"" + e.getMessage() + "\"");
                e.printStackTrace(System.err);
            }
            result = TransactionResultUtil.setNonRevertedFailureAndEnergyUsed(result, AvmInternalError.FAILED_OUT_OF_STACK, tx.energyLimit);

        } catch (CallDepthLimitExceededException e) {
            if (verboseErrors) {
                System.err.println("DApp deployment failed due to call depth limit EXCEPTION: \"" + e.getMessage() + "\"");
                e.printStackTrace(System.err);
            }
            result = TransactionResultUtil.setNonRevertedFailureAndEnergyUsed(result, AvmInternalError.FAILED_CALL_DEPTH_LIMIT, tx.energyLimit);

        } catch (RevertException e) {
            if (verboseErrors) {
                System.err.println("DApp deployment to REVERT due to uncaught EXCEPTION: \"" + e.getMessage() + "\"");
                e.printStackTrace(System.err);
            }
            result = TransactionResultUtil.setRevertedFailureAndEnergyUsed(result, tx.energyLimit);

        } catch (InvalidException e) {
            if (verboseErrors) {
                System.err.println("DApp deployment INVALID due to uncaught EXCEPTION: \"" + e.getMessage() + "\"");
                e.printStackTrace(System.err);
            }
            result = TransactionResultUtil.setNonRevertedFailureAndEnergyUsed(result, AvmInternalError.FAILED_INVALID, tx.energyLimit);

        } catch (UncaughtException e) {
            if (verboseErrors) {
                System.err.println("DApp deployment failed due to uncaught EXCEPTION: \"" + e.getMessage() + "\"");
                e.printStackTrace(System.err);
            }
            result = TransactionResultUtil.setFailedException(result, e.getCause(), tx.energyLimit);
        } catch (RejectedClassException e) {
            if (verboseErrors) {
                System.err.println("DApp deployment REJECTED with reason: \"" + e.getMessage() + "\"");
                e.printStackTrace(System.err);
            }
            result = TransactionResultUtil.setNonRevertedFailureAndEnergyUsed(result, AvmInternalError.FAILED_REJECTED_CLASS, tx.energyLimit);

        } catch (EarlyAbortException e) {
            if (verboseErrors) {
                System.err.println("FYI - concurrent abort (will retry) in transaction \"" + Helpers.bytesToHexString(tx.copyOfTransactionHash()) + "\"");
            }
            result = TransactionResultUtil.newAbortedResultWithZeroEnergyUsed();

        } catch (AvmException e) {
            // We handle the generic AvmException as some failure within the contract.
            if (verboseErrors) {
                System.err.println("DApp deployment failed due to AvmException: \"" + e.getMessage() + "\"");
                e.printStackTrace(System.err);
            }
            result = TransactionResultUtil.setNonRevertedFailureAndEnergyUsed(result, AvmInternalError.FAILED, tx.energyLimit);
        } catch (JvmError e) {
            // These are cases which we know we can't handle and have decided to handle by safely stopping the AVM instance so
            // re-throw this as the AvmImpl top-level loop will commute it into an asynchronous shutdown.
            if (verboseErrors) {
                System.err.println("FATAL JvmError: \"" + e.getMessage() + "\"");
                e.printStackTrace(System.err);
            }
            throw e;
        } catch (Throwable e) {
            // We don't know what went wrong in this case, but it is beyond our ability to handle it here.
            // We ship it off to the ExceptionHandler, which kills the transaction as a failure for unknown reasons.
            result = DAppExceptionHandler.handle(e, result, tx.energyLimit, verboseErrors);
        } finally {
            // Once we are done running this, no matter how it ended, we want to detach our thread from the DApp.
            if (null != dapp) {
                InstrumentationHelpers.popExistingStackFrame(dapp.runtimeSetup);
            }
        }
        return result;
    }

    public static Map<String, byte[]> stripClinitFromClasses(Map<String, byte[]> transformedClasses){
        Map<String, byte[]> immortalClasses = new HashMap<>();
        for (Map.Entry<String, byte[]> elt : transformedClasses.entrySet()) {
            String className = elt.getKey();
            byte[] transformedClass = elt.getValue();
            byte[] immortalClass = new ClassToolchain.Builder(transformedClass, 0)
                    .addNextVisitor(new ClinitStrippingVisitor())
                    .addWriter(new ClassWriter(0))
                    .build()
                    .runAndGetBytecode();
            immortalClasses.put(className, immortalClass);
        }
        return immortalClasses;
    }

    private static Map<String, byte[]> rejectionAndRenameInputClasses(Map<String, byte[]> inputClasses, ClassHierarchy classHierarchy, ClassRenamer classRenamer, boolean preserveDebuggability) {
        // By this point, we at least know that the classHierarchy is internally consistent.
        // This also means we can safely count instance variables to make sure we haven't reached our limit.
        InstanceVariableCountManager manager = new InstanceVariableCountManager();
        Map<String, byte[]> safeClasses = new HashMap<>();

        Set<String> preRenameUserClassAndInterfaceSet = classHierarchy.getPreRenameUserDefinedClassesAndInterfaces();
        Set<String> preRenameUserDefinedClasses = classHierarchy.getPreRenameUserDefinedClassesOnly(classRenamer);

        PreRenameClassAccessRules preRenameClassAccessRules = new PreRenameClassAccessRules(preRenameUserDefinedClasses, preRenameUserClassAndInterfaceSet);
        NamespaceMapper namespaceMapper = new NamespaceMapper(preRenameClassAccessRules);
        
        for (String name : inputClasses.keySet()) {
            // Note that transformClasses requires that the input class names by the .-style names.
            RuntimeAssertionError.assertTrue(-1 == name.indexOf("/"));

            int parsingOptions = preserveDebuggability ? 0: ClassReader.SKIP_DEBUG;
            try {
                InstanceVariableCountingVisitor variableCounter = new InstanceVariableCountingVisitor(manager);
                byte[] bytecode = new ClassToolchain.Builder(inputClasses.get(name), parsingOptions)
                    .addNextVisitor(new RejectionClassVisitor(preRenameClassAccessRules, namespaceMapper, preserveDebuggability))
                    .addNextVisitor(new LoopingExceptionStrippingVisitor())
                    .addNextVisitor(variableCounter)
                    .addNextVisitor(new UserClassMappingVisitor(namespaceMapper, preserveDebuggability))
                    .addWriter(new TypeAwareClassWriter(ClassWriter.COMPUTE_FRAMES | ClassWriter.COMPUTE_MAXS, classHierarchy, classRenamer))
                    .build()
                    .runAndGetBytecode();
                String mappedName = DebugNameResolver.getUserPackageDotPrefix(name, preserveDebuggability);
                safeClasses.put(mappedName, bytecode);
            } catch (Exception e) {
                throw new RejectedClassException(e.getMessage());
            }
        }
        // Before we return, make sure we didn't exceed the instance variable limits (will throw RejectedClassException on failure).
        manager.verifyAllCounts();
        return safeClasses;
    }
}
