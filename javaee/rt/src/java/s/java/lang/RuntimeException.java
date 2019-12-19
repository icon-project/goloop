package s.java.lang;

import i.IInstrumentation;


/**
 * Our shadow implementation of java.lang.RuntimeException.
 * 
 * This only exists as an intermediary since we needed to implement a few specific subclasses.
 */
public class RuntimeException extends Exception {
    static {
        // Shadow classes MUST be loaded during bootstrap phase.
        IInstrumentation.attachedThreadInstrumentation.get().bootstrapOnly();
    }

    public RuntimeException() {
        super();
    }

    public RuntimeException(String message) {
        super(message);
    }

    public RuntimeException(String message, Throwable cause) {
        super(message, cause);
    }

    public RuntimeException(Throwable cause) {
        super(cause);
    }

    // Deserializer support.
    public RuntimeException(java.lang.Void ignore, int readIndex) {
        super(ignore, readIndex);
    }
}
