package s.java.lang;

import i.IInstrumentation;


/**
 * Our shadow implementation of java.lang.Error.
 * 
 * This only exists as an intermediary since we needed to implement AssertionError.
 */
public class Error extends Throwable {
    static {
        // Shadow classes MUST be loaded during bootstrap phase.
        IInstrumentation.attachedThreadInstrumentation.get().bootstrapOnly();
    }

    public Error() {
        super();
    }

    public Error(String message) {
        super(message);
    }

    public Error(String message, Throwable cause) {
        super(message, cause);
    }

    public Error(Throwable cause) {
        super(cause);
    }

    // Deserializer support.
    public Error(java.lang.Void ignore, int readIndex) {
        super(ignore, readIndex);
    }
}
