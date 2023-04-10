package foundation.icon.ee.score;

import foundation.icon.ee.Agent;
import foundation.icon.ee.types.IllegalFormatException;
import foundation.icon.ee.types.Status;
import i.GenericPredefinedException;
import i.PackageConstants;
import i.RuntimeAssertionError;
import org.aion.avm.core.AvmConfiguration;
import org.aion.avm.core.ClassHierarchyForest;
import org.aion.avm.core.ClassRenamer;
import org.aion.avm.core.ClassToolchain;
import org.aion.avm.core.ConstantClassBuilder;
import org.aion.avm.core.IExternalState;
import org.aion.avm.core.NodeEnvironment;
import org.aion.avm.core.TypeAwareClassWriter;
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
import org.aion.avm.core.verification.Verifier;
import org.aion.avm.utilities.JarBuilder;
import org.aion.avm.utilities.Utilities;
import org.aion.avm.utilities.analyze.ClassFileInfoBuilder;
import org.objectweb.asm.ClassReader;
import org.objectweb.asm.ClassWriter;

import java.io.IOException;
import java.util.HashMap;
import java.util.Map;
import java.util.Set;
import java.util.zip.ZipException;

public class Transformer {
    /**
     * Returns the sizes of all the user-space classes
     *
     * @param classHierarchy     the class hierarchy
     * @return The look-up map of the sizes of user objects
     */
    private static Map<String, Integer> computeUserObjectSizes(Forest<String, ClassInfo> classHierarchy)
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
    private static Map<String, byte[]> transformClasses(Map<String, byte[]> inputClasses, Forest<String, ClassInfo> oldPreRenameForest, ClassHierarchy classHierarchy, ClassRenamer classRenamer, boolean preserveDebuggability) {
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
        // We also want to expose this type to the class writer, so it can compute common superclasses.
        GeneratedClassConsumer generatedClassesSink = (superClassSlashName, classSlashName, bytecode) -> {
            // Note that the processed classes are expected to use .-style names.
            String classDotName = Utilities.internalNameToFullyQualifiedName(classSlashName);
            processedClasses.put(classDotName, bytecode);
        };
        Map<String, Integer> postRenameObjectSizes = computeAllPostRenameObjectSizes(oldPreRenameForest, preserveDebuggability);

        Map<String, byte[]> transformedClasses = new HashMap<>();

        int parsingOptions = preserveDebuggability ? ClassReader.EXPAND_FRAMES : ClassReader.EXPAND_FRAMES | ClassReader.SKIP_DEBUG;

        for (String name : safeClasses.keySet()) {
            // Note that transformClasses requires that the input class names by the .-style names.
            RuntimeAssertionError.assertTrue(!name.contains("/"));

            // We need to parse with EXPAND_FRAMES, since the StackWatcherClassAdapter uses a MethodNode to parse methods.
            // We also add SKIP_DEBUG since we aren't using debug data and skipping it removes extraneous labels which would otherwise
            // cause the BlockBuildingMethodVisitor to build lots of small blocks instead of a few big ones (each block incurs a Helper
            // static call, which is somewhat expensive - this is how we bill for energy).
            var builder = new ClassToolchain.Builder(safeClasses.get(name),
                    parsingOptions);
            Agent agent = Agent.get();
            if (agent == null || agent.isClassMeteringEnabled()) {
                builder.addNextVisitor(new ClassMetering(postRenameObjectSizes));
            }

            byte[] bytecode = builder.addNextVisitor(new ConstantVisitor(PackageConstants.kConstantClassName, constantClass.constantToFieldMap))
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

    private static Map<String, byte[]> stripClinitFromClasses(Map<String, byte[]> transformedClasses){
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
            RuntimeAssertionError.assertTrue(!name.contains("/"));

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

    private final IExternalState es;
    private final AvmConfiguration conf;
    private byte[] transformedCodeBytes;
    private byte[] apisBytes;
    private TransformedDappModule bootstrapModule;

    public Transformer(IExternalState es, AvmConfiguration conf) {
        this.es = es;
        this.conf = conf;
    }

    public void transform() {
        try {
            transformImpl();
        } catch (ZipException e) {
            throw new GenericPredefinedException(Status.PackageError, e);
        } catch (IOException e) {
            throw new IllegalFormatException(e);
        }
    }

    private void transformImpl() throws IOException {
        byte[] codeBytes = es.getCode();
        apisBytes = JarBuilder.getAPIsBytesFromJAR(codeBytes);
        if (apisBytes == null) {
            throw new IllegalFormatException("bad APIS");
        }

        RawDappModule rawDapp = RawDappModule.readFromJar(codeBytes,
                    conf.preserveDebuggability);
        if (!rawDapp.classes.containsKey(rawDapp.mainClass)) {
            throw new IllegalFormatException("no main class");
        }
        ClassHierarchyForest dappClassesForest = rawDapp.classHierarchyForest;

        // transform
        Map<String, byte[]> transformedClasses = transformClasses(
                rawDapp.classes, dappClassesForest, rawDapp.classHierarchy,
                rawDapp.classRenamer, conf.preserveDebuggability);
        bootstrapModule = TransformedDappModule.fromTransformedClasses(transformedClasses, rawDapp.mainClass);
        Map<String, byte[]> immortalClasses = stripClinitFromClasses(transformedClasses);
        ImmortalDappModule immortalDapp = ImmortalDappModule.fromImmortalClasses(immortalClasses, bootstrapModule.mainClass, apisBytes);
        transformedCodeBytes = immortalDapp.createJar(es.getBlockTimestamp());
    }

    public TransformedDappModule getBootstrapModule() {
        return bootstrapModule;
    }

    public byte[] getTransformedCodeBytes() {
        return transformedCodeBytes;
    }

    public byte[] getAPIsBytes() {
        return apisBytes;
    }
}
