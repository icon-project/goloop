package s.java.lang;

import i.IInstrumentation;
import i.IObject;


/**
 * Our shadow implementation of java.lang.AssertionError.
 * 
 * This requires manual implementation since it has many constructors.
 */
public class AssertionError extends Error {
    static {
        // Shadow classes MUST be loaded during bootstrap phase.
        IInstrumentation.attachedThreadInstrumentation.get().bootstrapOnly();
    }

    public AssertionError() {
    }

    private AssertionError(String detailMessage) {
        super(detailMessage);
    }

    public AssertionError(IObject detailMessage) {
        this(String.avm_valueOf((Object)detailMessage), (detailMessage instanceof Throwable) ? (Throwable) detailMessage : null);
    }

    public AssertionError(boolean detailMessage) {
        this(String.avm_valueOf(detailMessage));
    }

    public AssertionError(char detailMessage) {
        this(String.avm_valueOf(detailMessage));
    }

    public AssertionError(int detailMessage) {
        this(String.avm_valueOf(detailMessage));
    }

    public AssertionError(long detailMessage) {
        this(String.avm_valueOf(detailMessage));
    }

    public AssertionError(float detailMessage) {
        this(String.avm_valueOf(detailMessage));
    }

    public AssertionError(double detailMessage) {
        this(String.avm_valueOf(detailMessage));
    }

    public AssertionError(String message, Throwable cause) {
        super(message, cause);
    }

    // Deserializer support.
    public AssertionError(java.lang.Void ignore, int readIndex) {
        super(ignore, readIndex);
    }
}
