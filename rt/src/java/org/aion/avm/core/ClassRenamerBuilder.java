package org.aion.avm.core;

import java.util.HashSet;
import java.util.Set;
import org.aion.avm.NameStyle;
import org.aion.avm.core.ClassRenamer.ClassCategory;
import org.aion.avm.core.ClassRenamer.NameCategory;

public final class ClassRenamerBuilder {
    private final boolean preserveDebuggability;
    private final NameStyle style;
    private Set<String> jclExceptions = new HashSet<>();
    private NameCategory jclExceptionsCategory = NameCategory.PRE_RENAME;
    private Set<String> userDefinedClasses = new HashSet<>();
    private NameCategory userDefinedClassesCategory = NameCategory.PRE_RENAME;
    private Set<ClassCategory> prohibitedClasses = new HashSet<>();

    public ClassRenamerBuilder(NameStyle style, boolean preserveDebuggability) {
        this.style = style;
        this.preserveDebuggability = preserveDebuggability;
    }

    public ClassRenamerBuilder loadPreRenameJclExceptionClasses(Set<String> jclExceptions) {
        this.jclExceptions = jclExceptions;
        this.jclExceptionsCategory = NameCategory.PRE_RENAME;
        return this;
    }

    public ClassRenamerBuilder loadPostRenameJclExceptionClasses(Set<String> jclExceptions) {
        this.jclExceptions = jclExceptions;
        this.jclExceptionsCategory = NameCategory.POST_RENAME;
        return this;
    }

    public ClassRenamerBuilder loadPreRenameUserDefinedClasses(Set<String> userDefinedClasses) {
        this.userDefinedClasses = userDefinedClasses;
        this.userDefinedClassesCategory = NameCategory.PRE_RENAME;
        return this;
    }

    public ClassRenamerBuilder loadPostRenameUserDefinedClasses(Set<String> userDefinedClasses) {
        this.userDefinedClasses = userDefinedClasses;
        this.userDefinedClassesCategory = NameCategory.POST_RENAME;
        return this;
    }

    public ClassRenamerBuilder prohibitJclClasses() {
        this.prohibitedClasses.add(ClassCategory.JCL);
        return this;
    }

    public ClassRenamerBuilder prohibitApiClasses() {
        this.prohibitedClasses.add(ClassCategory.API);
        return this;
    }

    public ClassRenamerBuilder prohibitExceptionWrappers() {
        this.prohibitedClasses.add(ClassCategory.EXCEPTION_WRAPPER);
        return this;
    }

    public ClassRenamerBuilder prohibitPreciseArrayTypes() {
        this.prohibitedClasses.add(ClassCategory.PRECISE_ARRAY);
        return this;
    }

    public ClassRenamerBuilder prohibitUnifyingArrayTypes() {
        this.prohibitedClasses.add(ClassCategory.UNIFYING_ARRAY);
        return this;
    }

    public ClassRenamerBuilder prohibitUserClasses() {
        this.prohibitedClasses.add(ClassCategory.USER);
        return this;
    }

    public ClassRenamer build() {
        return new ClassRenamer(
            this.preserveDebuggability,
            this.style,
            this.jclExceptions,
            this.jclExceptionsCategory,
            this.userDefinedClasses,
            this.userDefinedClassesCategory,
            this.prohibitedClasses);
    }

}
