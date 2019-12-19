package s.java.lang;

import i.IInstrumentation;


/**
 * Our shadow implementation of java.lang.Exception.
 * 
 * This only exists as an intermediary since we needed to implement a few specific subclasses.
 */
public class Exception extends Throwable {
    static {
        // Shadow classes MUST be loaded during bootstrap phase.
        IInstrumentation.attachedThreadInstrumentation.get().bootstrapOnly();
    }

    public Exception() {
        super();
    }

    public Exception(String message) {
        super(message);
    }

    public Exception(String message, Throwable cause) {
        super(message, cause);
    }

    public Exception(Throwable cause) {
        super(cause);
    }

    // Deserializer support.
    public Exception(java.lang.Void ignore, int readIndex) {
        super(ignore, readIndex);
    }
}
