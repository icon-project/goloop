package s.java.lang;

import i.IInstrumentation;


/**
 * Our shadow implementation of java.lang.TypeNotPresentException.
 */
public class TypeNotPresentException extends RuntimeException {
    static {
        // Shadow classes MUST be loaded during bootstrap phase.
        IInstrumentation.attachedThreadInstrumentation.get().bootstrapOnly();
    }

    private String typeName;

    public TypeNotPresentException(String typeName, Throwable cause) {
        super(new String("Type " + typeName + " not present"), cause);
        this.typeName = typeName;
    }

    public TypeNotPresentException(java.lang.Void ignore, int readIndex) {
        super(ignore, readIndex);
    }

    //=======================================================
    // Methods below are used by runtime and test code only!
    //========================================================

    public String typeName() {
        lazyLoad();
        return typeName;
    }
}
