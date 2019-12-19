package org.aion.avm.core.types;

import java.util.HashSet;
import java.util.Map;

import java.util.Set;
import org.aion.avm.NameStyle;
import org.aion.avm.core.ClassHierarchyForest;
import org.aion.avm.core.ClassRenamer;
import org.aion.avm.core.ClassRenamerBuilder;
import org.aion.avm.core.dappreading.LoadedJar;
import org.aion.avm.core.rejection.RejectedClassException;


/**
 * Represents the original code submitted by the user, prior to validation or transformation.
 * Once transformed, the code this contains is moved into a TransformedDappModule.
 * All fields are public since this object is effectively an immutable struct.
 * See issue-134 for more details on this design.
 */
public class RawDappModule {
    /**
     * Reads the Dapp module from JAR bytes, in memory.
     * Note that a Dapp module is expected to specify a main class and contain at least one class.
     * 
     * @param jar The JAR bytes.
     * @param preserveDebuggability True if debug data within the JAR should be preserved.
     * @param verboseErrors True if the underlying reason for the deployment failure should be logged (typically for corrupt data).
     * @return The module, or null if the contents of the JAR were insufficient for a Dapp.
     */
    public static RawDappModule readFromJar(byte[] jar, boolean preserveDebuggability, boolean verboseErrors) {
        // Note that ASM can fail with all kinds of exceptions so we will handle any exception as an error.
        try {
            LoadedJar loadedJar = LoadedJar.fromBytes(jar);
            ClassHierarchyForest forest = ClassHierarchyForest.createForestFrom(loadedJar);

            // Construct the complete class hierarchy.
            ClassInformationFactory classInfoFactory = new ClassInformationFactory();
            Set<ClassInformation> classInfos = classInfoFactory.fromUserDefinedPreRenameJar(loadedJar);

            ClassRenamer classRenamer = new ClassRenamerBuilder(NameStyle.DOT_NAME, preserveDebuggability)
                .loadPreRenameUserDefinedClasses(extractClassNames(classInfos))
                .loadPostRenameJclExceptionClasses(fetchPostRenameJclExceptions())
                .build();

            ClassHierarchy fullHierarchy = new ClassHierarchyBuilder()
                .addShadowJcl()
                .addPreRenameUserDefinedClasses(classRenamer, classInfos)
                .addHandwrittenArrayWrappers()
                .addPostRenameJclExceptions()
                .build();

            Map<String, byte[]> classes = loadedJar.classBytesByQualifiedNames;
            String mainClass = loadedJar.mainClassName;
            // To be a valid Dapp, this must specify a main class and have at least one class.
            return ((null != mainClass) && !classes.isEmpty())
                ? new RawDappModule(classes, mainClass, forest, jar.length, classes.size(), fullHierarchy, classRenamer)
                : null;
        } catch (RejectedClassException e) {
            throw e;
        } catch (Throwable t) {
            // Since this can fail for myriad of reasons, we do not re-throw exceptions here.
            // null will be interpreted as a malformed dapp jar by DappCreator and an FAILED_INVALID_DATA exception will be thrown.
            if (verboseErrors) {
                System.err.println("Reading dapp jar bytes failed.");
                t.printStackTrace();
            }
            return null;
        }
    }

    private static Set<String> extractClassNames(Set<ClassInformation> classInformations) {
        Set<String> classNames = new HashSet<>();
        for (ClassInformation classInformation : classInformations) {
            classNames.add(classInformation.dotName);
        }
        return classNames;
    }

    private static Set<String> fetchPostRenameJclExceptions() {
        Set<String> exceptions = new HashSet<>();
        for (CommonType type : CommonType.values()) {
            if (type.isShadowException) {
                exceptions.add(type.dotName);
            }
        }
        return exceptions;
    }

    public final Map<String, byte[]> classes;
    public final String mainClass;
    public final ClassHierarchyForest classHierarchyForest;
    public final ClassHierarchy classHierarchy;
    public final ClassRenamer classRenamer;

    // For billing purpose
    public final long bytecodeSize;
    public final long numberOfClasses;
    
    private RawDappModule(Map<String, byte[]> classes, String mainClass, ClassHierarchyForest classHierarchyForest, long bytecodeSize, long numberOfClasses, ClassHierarchy hierarchy, ClassRenamer classRenamer) {
        this.classes = classes;
        this.mainClass = mainClass;
        this.classHierarchyForest = classHierarchyForest;
        this.bytecodeSize = bytecodeSize;
        this.numberOfClasses = numberOfClasses;
        this.classHierarchy = hierarchy;
        this.classRenamer = classRenamer;
    }
}
