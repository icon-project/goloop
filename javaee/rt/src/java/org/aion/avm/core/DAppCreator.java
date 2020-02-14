package org.aion.avm.core;

import foundation.icon.ee.types.Address;
import foundation.icon.ee.types.Result;
import foundation.icon.ee.types.Status;
import foundation.icon.ee.types.CodedException;
import foundation.icon.ee.types.Transaction;
import i.AvmException;
import i.CallDepthLimitExceededException;
import i.EarlyAbortException;
import i.GenericCodedException;
import i.IBlockchainRuntime;
import i.IInstrumentation;
import i.IRuntimeSetup;
import i.InstrumentationHelpers;
import i.JvmError;
import i.OutOfStackException;
import i.PackageConstants;
import i.RuntimeAssertionError;
import org.aion.avm.RuntimeMethodFeeSchedule;
import org.aion.avm.StorageFees;
import org.aion.avm.core.arraywrapping.ArraysRequiringAnalysisClassVisitor;
import org.aion.avm.core.arraywrapping.ArraysWithKnownTypesClassVisitor;
import org.aion.avm.core.exceptionwrapping.ExceptionWrapping;
import org.aion.avm.core.instrument.ClassMetering;
import org.aion.avm.core.instrument.HeapMemoryCostCalculator;
import org.aion.avm.core.miscvisitors.APIRemapClassVisitor;
import org.aion.avm.core.miscvisitors.ClinitStrippingVisitor;
import org.aion.avm.core.miscvisitors.ConstantVisitor;
import org.aion.avm.core.miscvisitors.InterfaceFieldClassGeneratorVisitor;
import org.aion.avm.core.miscvisitors.InterfaceFieldNameMappingVisitor;
import org.aion.avm.core.miscvisitors.LoopingExceptionStrippingVisitor;
import org.aion.avm.core.miscvisitors.NamespaceMapper;
import org.aion.avm.core.miscvisitors.PreRenameClassAccessRules;
import org.aion.avm.core.miscvisitors.StrictFPVisitor;
import org.aion.avm.core.miscvisitors.UserClassMappingVisitor;
import org.aion.avm.core.persistence.AutomaticGraphVisitor;
import org.aion.avm.core.persistence.LoadedDApp;
import org.aion.avm.core.rejection.ConsensusLimitConstants;
import org.aion.avm.core.rejection.InstanceVariableCountManager;
import org.aion.avm.core.rejection.InstanceVariableCountingVisitor;
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
import org.aion.avm.utilities.JarBuilder;
import org.aion.avm.utilities.Utilities;
import org.aion.avm.utilities.analyze.ClassFileInfoBuilder;
import org.aion.parallel.TransactionTask;
import org.objectweb.asm.ClassReader;
import org.objectweb.asm.ClassWriter;

import java.util.HashMap;
import java.util.Map;
import java.util.Set;

public class DAppCreator {
    /**
     * Returns the sizes of all the user-space classes
     *
     * @param classHierarchy     the class hierarchy
     * @return The look-up map of the sizes of user objects
     * Class name is in the JVM internal name format, see {@link org.aion.avm.core.util.Helpers#fulllyQualifiedNameToInternalName(String)}
     */
    public static Map<String, Integer> computeUserObjectSizes(Forest<String, ClassInfo> classHierarchy)
    {
        HeapMemoryCostCalculator objectSizeCalculator = new HeapMemoryCostCalculator();

        // compute the user object sizes
        objectSizeCalculator.calcClassesInstanceSize(classHierarchy);

        return objectSizeCalculator.getClassHeapSizeMap();
    }

    private static Map<String, Integer> computeAllPostRenameObjectSizes(Forest<String, ClassInfo> forest, boolean preserveDebuggability) {
        Map<String, Integer> preRenameUserObjectSizes = computeUserObjectSizes(forest);

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
            String classDotName = Utilities.internalNameToFulllyQualifiedName(classSlashName);
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
                    .addNextVisitor(new ArraysRequiringAnalysisClassVisitor(classHierarchy))
                    .addNextVisitor(new ArraysWithKnownTypesClassVisitor())
                    .addNextVisitor(new APIRemapClassVisitor())
                    .addWriter(new TypeAwareClassWriter(ClassWriter.COMPUTE_FRAMES | ClassWriter.COMPUTE_MAXS, classHierarchy, classRenamer))
                    .build()
                    .runAndGetBytecode();
            transformedClasses.put(name, bytecode);
        }

        /*
         * Another pass to deal with static fields in interfaces.
         * Note that all fields in interfaces are defined as static.
         */
        // mapping between interface name and generated class name containing all the interface fields
        Map<String, String> interfaceFieldClassNames = new HashMap<>();

        String javaLangObjectSlashName = PackageConstants.kShadowSlashPrefix + "java/lang/Object";
        for (String name : transformedClasses.keySet()) {
            // This visitor does not modify the byte code of transformedClasses. It only generates a new class containing fields and clinit for each interface.
            new ClassReader(transformedClasses.get(name))
                    .accept(new InterfaceFieldClassGeneratorVisitor(generatedClassesSink, interfaceFieldClassNames, javaLangObjectSlashName), parsingOptions);
        }

        for (String name : transformedClasses.keySet()) {
            byte[] bytecode = new ClassToolchain.Builder(transformedClasses.get(name), parsingOptions)
                    .addNextVisitor(new InterfaceFieldNameMappingVisitor(interfaceFieldClassNames))
                    .addWriter(new TypeAwareClassWriter(ClassWriter.COMPUTE_FRAMES | ClassWriter.COMPUTE_MAXS, classHierarchy, classRenamer))
                    .build()
                    .runAndGetBytecode();

            processedClasses.put(name, bytecode);
        }

        return processedClasses;
    }

    public static Result create(IExternalState externalState,
                                                     TransactionTask task,
                                                     Address senderAddress,
                                                     Address dappAddress,
                                                     Transaction tx,
                                                     long energyPreused,
                                                     boolean preserveDebuggability,
                                                     boolean verboseErrors,
                                                     boolean enableBlockchainPrintln) {
        // We hold onto the runtimeSetup that we are pushing onto the stack in here so that we can pop it back off in the finally block.
        IRuntimeSetup runtimeSetup = null;
        Result result = null;
        try {
            final byte[] codeBytes = externalState.getCode(dappAddress);
            byte[] apisBytes = JarBuilder.getAPIsBytesFromJAR(codeBytes);
            if (apisBytes == null) {
                if (verboseErrors) {
                    System.err.println("DApp deployment failed due to bad external methods info");
                }
                return new Result(Status.IllegalFormat, tx.getLimit(), "bad external method info");
            }

            RawDappModule rawDapp = RawDappModule.readFromJar(codeBytes, preserveDebuggability, verboseErrors);
            if (rawDapp == null) {
                if (verboseErrors) {
                    System.err.println("DApp deployment failed due to corrupt JAR data");
                }
                return new Result(Status.IllegalFormat, tx.getLimit(), "bad jar data");
            }

            // Verify that the DApp contains the main class they listed and that it has a "public static byte[] main()" method.
            if (!rawDapp.classes.containsKey(rawDapp.mainClass)) {
                if (verboseErrors) {
                    String explanation = "missing Main class";
                    System.err.println("DApp deployment failed due to " + explanation);
                }
                return new Result(Status.IllegalFormat, tx.getLimit(), "missing Main class");
            }
            ClassHierarchyForest dappClassesForest = rawDapp.classHierarchyForest;

            // transform
            Map<String, byte[]> transformedClasses = transformClasses(rawDapp.classes, dappClassesForest, rawDapp.classHierarchy, rawDapp.classRenamer, preserveDebuggability);
            TransformedDappModule transformedDapp = TransformedDappModule.fromTransformedClasses(transformedClasses, rawDapp.mainClass);

            LoadedDApp dapp = DAppLoader.fromTransformed(transformedDapp, apisBytes, preserveDebuggability);
            try {
                dapp.verifyExternalMethods();
            } catch (Exception e) {
                if (verboseErrors) {
                    String explanation = "missing External methods";
                    System.err.println("DApp deployment failed due to " + explanation + " exception:" + e);
                    e.printStackTrace();
                }
                return new Result(Status.IllegalFormat, tx.getLimit(), e.toString());
            }
            runtimeSetup = dapp.runtimeSetup;

            // We start the nextHashCode at 1.
            int nextHashCode = 1;
            // we pass a null re-entrant state since we haven't finished initializing yet - nobody can call into us.
            IBlockchainRuntime br = new BlockchainRuntimeImpl(externalState,
                                                              task,
                                                              senderAddress,
                                                              dappAddress,
                                                              tx,
                                                              runtimeSetup,
                                                              dapp,
                                                              enableBlockchainPrintln);
            FrameContextImpl fc = new FrameContextImpl(externalState, dapp, dapp.getInternedClasses(), br);
            InstrumentationHelpers.pushNewStackFrame(runtimeSetup, dapp.loader, tx.getLimit() - energyPreused, nextHashCode, dapp.getInternedClasses(), fc);
            IBlockchainRuntime previousRuntime = dapp.attachBlockchainRuntime(br);

            // We have just created this dApp, there should be no previous runtime associated with it.
            RuntimeAssertionError.assertTrue(previousRuntime == null);

            IInstrumentation threadInstrumentation = IInstrumentation.attachedThreadInstrumentation.get();
            long deploymentFee = BillingRules.getDeploymentFee(rawDapp.numberOfClasses, rawDapp.bytecodeSize);
            // Deployment fee must be a positive integer.
            RuntimeAssertionError.assertTrue(deploymentFee > 0L);
            RuntimeAssertionError.assertTrue(deploymentFee <= (long)Integer.MAX_VALUE);
            threadInstrumentation.chargeEnergy((int)deploymentFee);

            // Create the immortal version of the transformed DApp code by stripping the <clinit>.
            Map<String, byte[]> immortalClasses = stripClinitFromClasses(transformedClasses);

            ImmortalDappModule immortalDapp = ImmortalDappModule.fromImmortalClasses(immortalClasses, transformedDapp.mainClass, apisBytes);

            // store transformed dapp
            byte[] immortalDappJar = immortalDapp.createJar(externalState.getBlockTimestamp());
            externalState.setTransformedCode(dappAddress, immortalDappJar);

            // Force the classes in the dapp to initialize so that the <clinit> is run (since we already saved the version without).
            result = runClinitAndBillSender(verboseErrors, dapp, threadInstrumentation, externalState, task, dappAddress, tx);
        } catch (CodedException e) {
            if (verboseErrors) {
                System.err.println("DApp deployment to REVERT due to uncaught EXCEPTION: \"" + e.getMessage() + "\"");
                e.printStackTrace(System.err);
            }
            result = new Result(e.getCode(),
                    tx.getLimit() - IInstrumentation.getEnergyLeft(),
                    e.toString());

        } catch (EarlyAbortException e) {
            if (verboseErrors) {
                System.err.println("FYI - concurrent abort (will retry) in transaction \"" + Helpers.bytesToHexString(tx.copyOfTransactionHash()) + "\"");
            }
            assert false : "unexpected abort";

        } catch (AvmException e) {
            // We handle the generic AvmException as some failure within the contract.
            if (verboseErrors) {
                System.err.println("DApp deployment failed due to AvmException: \"" + e.getMessage() + "\"");
                e.printStackTrace(System.err);
            }
            result = new Result(Status.UnknownFailure, tx.getLimit(), e.toString());
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
            result = new Result(Status.UnknownFailure, tx.getLimit(), e.toString());
        } finally {
            // Once we are done running this, no matter how it ended, we want to detach our thread from the DApp.
            if (null != runtimeSetup) {
                InstrumentationHelpers.popExistingStackFrame(runtimeSetup);
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
                byte[] classBytecode = inputClasses.get(name);
                // Read the class to check our static geometry limits before running this through our high-level ASM rejection pipeline.
                // (note that this processing is done for HistogramDataCollector, back in AvmImpl, but this duplication isn't a large concern since that is disabled, by default).
                ClassFileInfoBuilder.ClassFileInfo classFileInfo = ClassFileInfoBuilder.getDirectClassFileInfo(classBytecode);

                // Impose class-level restrictions.
                if (classFileInfo.definedMethods.size() > ConsensusLimitConstants.MAX_METHOD_COUNT) {
                    throw RejectedClassException.maximumMethodCountExceeded(name);
                }
                if (classFileInfo.constantPoolEntryCount > ConsensusLimitConstants.MAX_CONSTANT_POOL_ENTRIES) {
                    throw RejectedClassException.maximumConstantPoolEntriesExceeded(name);
                }

                // Impose method-level restrictions.
                for (ClassFileInfoBuilder.MethodCode methodCode : classFileInfo.definedMethods) {
                    if (methodCode.codeLength > ConsensusLimitConstants.MAX_METHOD_BYTE_LENGTH) {
                        throw RejectedClassException.maximumMethodSizeExceeded(name);
                    }
                    if (methodCode.exceptionTableSize > ConsensusLimitConstants.MAX_EXCEPTION_TABLE_ENTRIES) {
                        throw RejectedClassException.maximumExceptionTableEntriesExceeded(name);
                    }
                    if (methodCode.maxStack > ConsensusLimitConstants.MAX_OPERAND_STACK_DEPTH) {
                        throw RejectedClassException.maximumOperandStackDepthExceeded(name);
                    }
                    if (methodCode.maxLocals > ConsensusLimitConstants.MAX_LOCAL_VARIABLES) {
                        throw RejectedClassException.maximumLocalVariableCountExceeded(name);
                    }
                }

                // Now, proceed with the ASM pipeline for high-level rejection and renaming.
                InstanceVariableCountingVisitor variableCounter = new InstanceVariableCountingVisitor(manager);
                byte[] bytecode = new ClassToolchain.Builder(classBytecode, parsingOptions)
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

    /**
     * Initializes all of the classes in the dapp by running their clinit code and then bills the
     * sender for writing the create data to the blockchain and refunds them accordingly.
     *
     * This method handles the following exceptions and ensures that if any of them are thrown
     * that they will be represented by the returned result (any other exceptions thrown here will
     * not be handled):
     * {@link OutOfStackException}, {@link CallDepthLimitExceededException}, and {@link GenericCodedException}.
     *
     * @param verboseErrors Whether or not to report errors to stderr.
     * @param dapp The dapp to run.
     * @param threadInstrumentation The thread instrumentation.
     * @param externalState The state of the world.
     * @param task The transaction task.
     * @param dappAddress The address of the contract.
     * @param tx The transaction.
     * @return the result of initializing and billing the sender.
     */
    private static Result runClinitAndBillSender(boolean verboseErrors,
                                                 LoadedDApp dapp,
                                                 IInstrumentation threadInstrumentation,
                                                 IExternalState externalState,
                                                 TransactionTask task,
                                                 Address dappAddress,
                                                 Transaction tx) throws Throwable {
        long energyLimit = tx.getLimit();
        Result resultToReturn;

        try {
            dapp.forceInitializeAllClasses();
            dapp.initMainInstance(tx.getParams());

            // Save back the state before we return.
            byte[] rawGraphData = dapp.saveEntireGraph(threadInstrumentation.peekNextHashCode(), StorageFees.MAX_GRAPH_SIZE);
            // Bill for writing this size.
            threadInstrumentation.chargeEnergy(StorageFees.WRITE_PRICE_PER_BYTE * rawGraphData.length);
            externalState.putObjectGraph(dappAddress, rawGraphData);

            long refund = 0;
            long energyUsed = energyLimit - threadInstrumentation.energyLeft();
            if (task.getTransactionStackDepth() == 0) {
                // refund is calculated for the transaction if it set the storage value from nonzero to zero
                long resetStorageRefund = 0L;

                if (task.getResetStorageKeyCount() > 0) {
                    resetStorageRefund = task.getResetStorageKeyCount() * RuntimeMethodFeeSchedule.BlockchainRuntime_avm_deleteStorage_refund;
                }
                // refund is capped at half the energy used for the whole transaction
                refund = Math.min(energyUsed / 2, resetStorageRefund);
            }
            // Return data of a CREATE transaction is the new DApp address.
            resultToReturn = new Result(Status.Success, energyUsed - refund, null);

        } catch (CodedException e) {
            if (verboseErrors) {
                System.err.println("DApp deployment failed due to stack overflow EXCEPTION: \"" + e.getMessage() + "\"");
                e.printStackTrace(System.err);
            }
            resultToReturn = new Result(e.getCode(),
                    energyLimit - threadInstrumentation.energyLeft(),
                    e.toString());
        }

        return resultToReturn;
    }
}
