package s.java.util;

import i.IInstrumentation;
import s.java.lang.RuntimeException;
import s.java.lang.String;


/**
 * Our shadow implementation of java.util.NoSuchElementException.
 * 
 * Implemented manually since it only provides a subset of the usual exception constructors.
 */
public class NoSuchElementException extends RuntimeException {
    static {
        // Shadow classes MUST be loaded during bootstrap phase.
        IInstrumentation.attachedThreadInstrumentation.get().bootstrapOnly();
    }

    public NoSuchElementException() {
        super();
    }

    public NoSuchElementException(String message) {
        super(message);
    }

    // Deserializer support.
    public NoSuchElementException(Void ignore, int readIndex) {
        super(ignore, readIndex);
    }
}
