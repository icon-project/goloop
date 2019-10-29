package org.aion.avm.core.rejection;


/**
 * Contains the constants used to impose limits on what the AVM is allowed to accept.
 */
public class ConsensusLimitConstants {
    /*
     * This will probably change, in the future, but we currently will only parse Java10 (version 54) classes.
     */
    public static final int SUPPORTED_CLASS_VERSION = 54;
    /*
     * This limit could probably be larger, since it really just needs to account for a type name length in 1 byte:
     * -probably 255 - 1 (for "L" prefix) - 3 (maximum array dimensions)
     * However, we want to leave this additional space for some potential future uses and we also don't expect that
     * these types will usually be longer than a few bytes long (API type references will probably be the longest)
     * because deployment tooling to shrink these identifiers reduces the deployment cost the user pays.
     */
    public static final int MAX_CLASS_NAME_UTF8_BYTES_LENGTH = 127;
    /*
     * We limit this to 31 since that avoids certain contrived cases where large objects can cause performance and billing problems.
     * (we only limit on the number of variables to keep the heuristic simple)
     * Note that this is the TOTAL instance variables across all super-classes, not just defined within the one class.
     */
    public static final int MAX_TOTAL_INSTANCE_VARIABLES = 31;
    /*
     * We impose a maximum code size to ensure that our bytecode instrumentation implementation details cannot be observed at the
     * level of consensus.
     * The JVM imposes a maximum length of 65535 bytes (with some special concerns around exception handlers at that size) so
     * capping this at 4095 means we have an overhead limit of 15 bytes per byte of input user code.
     * Given that a Java method larger than even 1KiB is quite rare, and that things need to be kept small for deployment, this
     * limit is unlikely to cause an issue.
     */
    public static final int MAX_METHOD_BYTE_LENGTH = 4095;
}
