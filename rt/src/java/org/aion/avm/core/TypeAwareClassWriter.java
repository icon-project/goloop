package org.aion.avm.core;

import org.aion.avm.core.types.ClassHierarchy;
import org.aion.avm.core.util.Helpers;
import i.RuntimeAssertionError;
import org.objectweb.asm.ClassWriter;


/**
 * We extend the ClassWriter to override their implementation of getCommonSuperClass() with an implementation which knows how
 * to compute this relationship between our generated classes, before they can be loaded.
 */
public class TypeAwareClassWriter extends ClassWriter {
    private final ArraySuperResolver arraySuperResolver;
    private final ExceptionWrapperSuperResolver exceptionWrapperSuperResolver;
    private final PlainTypeSuperResolver plainTypeSuperResolver;

    public TypeAwareClassWriter(int flags, ClassHierarchy hierarchy, ClassRenamer classRenamer) {
        super(flags);

        this.arraySuperResolver = new ArraySuperResolver(hierarchy, classRenamer);
        this.exceptionWrapperSuperResolver = new ExceptionWrapperSuperResolver(hierarchy, classRenamer);
        this.plainTypeSuperResolver = new PlainTypeSuperResolver(hierarchy, classRenamer);
    }

    @Override
    protected String getCommonSuperClass(String type1, String type2) {
        String type1dotName = Helpers.internalNameToFulllyQualifiedName(type1);
        String type2dotName = Helpers.internalNameToFulllyQualifiedName(type2);

        // We query the resolvers one by one if we get no valid response; we are guaranteed at least one will be non-null.
        String commonSuper = this.exceptionWrapperSuperResolver.getTightestSuperClassIfGivenPlainType(type1dotName, type2dotName);
        if (commonSuper != null) {
            return Helpers.fulllyQualifiedNameToInternalName(commonSuper);
        }

        // If the exception wrapper resolver couldn't find the super class we ask the array resolver.
        commonSuper = this.arraySuperResolver.getTightestSuperClassIfGivenArray(type1dotName, type2dotName);
        if (commonSuper != null) {
            return Helpers.fulllyQualifiedNameToInternalName(commonSuper);
        }

        // If we still have no answer we query the plain type resolver, which must give us an answer.
        commonSuper = this.plainTypeSuperResolver.getTightestSuperClassIfGivenPlainType(type1dotName, type2dotName);

        RuntimeAssertionError.assertTrue(commonSuper != null);
        return Helpers.fulllyQualifiedNameToInternalName(commonSuper);
    }
}
