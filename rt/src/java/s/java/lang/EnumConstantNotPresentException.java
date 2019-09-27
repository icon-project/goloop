package s.java.lang;

import i.IInstrumentation;


/**
 * Our shadow implementation of java.lang.EnumConstantNotPresentException.
 */
@SuppressWarnings("rawtypes") /* rawtypes are part of the public api */
public class EnumConstantNotPresentException extends RuntimeException {
    static {
        // Shadow classes MUST be loaded during bootstrap phase.
        IInstrumentation.attachedThreadInstrumentation.get().bootstrapOnly();
    }

    private Class<? extends Enum> enumType;
    private String constantName;

    public EnumConstantNotPresentException(Class<? extends Enum> enumType, String constantName) {
        super(new String(enumType.avm_getName() + "." + constantName));
        this.enumType = enumType;
        this.constantName  = constantName;
    }

    public EnumConstantNotPresentException(java.lang.Void ignore, int readIndex) {
        super(ignore, readIndex);
    }

    public String avm_constantName() {
        lazyLoad();
        return this.constantName;
    }

    public Class<? extends Enum> avm_enumType() {
        lazyLoad();
        return this.enumType;
    }
}
