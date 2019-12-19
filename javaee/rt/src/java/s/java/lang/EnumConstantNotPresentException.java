package s.java.lang;

import i.IInstrumentation;
import org.aion.avm.RuntimeMethodFeeSchedule;


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
        super(new String(enumType.getName() + "." + constantName));
        this.enumType = enumType;
        this.constantName  = constantName;
    }

    public EnumConstantNotPresentException(java.lang.Void ignore, int readIndex) {
        super(ignore, readIndex);
    }

    public String avm_constantName() {
        IInstrumentation.attachedThreadInstrumentation.get().chargeEnergy(RuntimeMethodFeeSchedule.Throwable_Hierarchy_Base_Fee);
        lazyLoad();
        return this.constantName;
    }

    public Class<? extends Enum> avm_enumType() {
        IInstrumentation.attachedThreadInstrumentation.get().chargeEnergy(RuntimeMethodFeeSchedule.Throwable_Hierarchy_Base_Fee);
        lazyLoad();
        return this.enumType;
    }
}
