package i;


/**
 * While most of the package constants (prefixes for our various namespaces, etc) are only used directly within the core module,
 * sometimes the Helper needs to know about them, as they represent part of the agreement between these 2 modules (for
 * instantiation, etc).
 */
public class PackageConstants {
    public static final String kShadowDotPrefix = "s.";
    public static final String kShadowApiDotPrefix = "p.";
    public static final String kExceptionWrapperDotPrefix = "e.";
    public static final String kArrayWrapperDotPrefix = "a.";
    public static final String kArrayWrapperUnifyingDotPrefix = "w.";
    public static final String kInternalDotPrefix = "i.";
    public static final String kUserDotPrefix = "u.";
    public static final String kPublicApiDotPrefix = "avm.";

    public static final String kShadowSlashPrefix = "s/";
    public static final String kShadowApiSlashPrefix = "p/";
    public static final String kExceptionWrapperSlashPrefix = "e/";
    public static final String kArrayWrapperSlashPrefix = "a/";
    public static final String kArrayWrapperUnifyingSlashPrefix = "w/";
    public static final String kInternalSlashPrefix = "i/";
    public static final String kUserSlashPrefix = "u/";
    public static final String kPublicApiSlashPrefix = "avm/";

    public static final String kConstantClassName = "C";
}
