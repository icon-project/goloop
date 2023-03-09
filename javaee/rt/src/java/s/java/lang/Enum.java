package s.java.lang;

import i.ConstantToken;
import i.IInstrumentation;
import i.IObject;
import org.aion.avm.RuntimeMethodFeeSchedule;
import s.java.io.Serializable;

public abstract class Enum<E extends Enum<E>> extends Object implements Comparable<E>, Serializable {
    static {
        // Shadow classes MUST be loaded during bootstrap phase.
        IInstrumentation.attachedThreadInstrumentation.get().bootstrapOnly();
    }

    // (note that these are not final since we want to be able to deserialize this type)
    private String name;
    private int ordinal;

    public final String avm_name() {
        IInstrumentation.attachedThreadInstrumentation.get().chargeEnergy(RuntimeMethodFeeSchedule.Enum_avm_name);
        return getName();
    }

    public final int avm_ordinal() {
        IInstrumentation.attachedThreadInstrumentation.get().chargeEnergy(RuntimeMethodFeeSchedule.Enum_avm_ordinal);
        lazyLoad();
        return ordinal;
    }

    protected Enum(String name, int ordinal) {
        IInstrumentation.attachedThreadInstrumentation.get().chargeEnergy(RuntimeMethodFeeSchedule.Enum_avm_constructor);
        this.name = name;
        this.ordinal = ordinal;
    }

    protected Enum(String name, int ordinal, ConstantToken constantToken) {
        super(constantToken);
        this.name = name;
        this.ordinal = ordinal;
    }

    // Deserializer support.
    public Enum(java.lang.Void ignore, int readIndex) {
        super(ignore, readIndex);
    }

    public String avm_toString() {
        IInstrumentation.attachedThreadInstrumentation.get().chargeEnergy(RuntimeMethodFeeSchedule.Enum_avm_toString);
        lazyLoad();
        return name;
    }

    public final boolean avm_equals(IObject other) {
        IInstrumentation.attachedThreadInstrumentation.get().chargeEnergy(RuntimeMethodFeeSchedule.Enum_avm_equals);
        lazyLoad();
        return this == other;
    }

    @Override
    public final Object avm_clone() throws CloneNotSupportedException {
        IInstrumentation.attachedThreadInstrumentation.get().chargeEnergy(RuntimeMethodFeeSchedule.Enum_avm_clone);
        throw new CloneNotSupportedException();
    }

    @Override
    public final int avm_compareTo(E o) {
        IInstrumentation.attachedThreadInstrumentation.get().chargeEnergy(RuntimeMethodFeeSchedule.Enum_avm_compareTo);
        lazyLoad();
        o.lazyLoad();
        Enum<?> other = o;
        Enum<E> self = this;
        if (self.avm_getClass() != other.avm_getClass() &&
                self.getDeclaringClass() != other.getDeclaringClass())
            throw new ClassCastException();
        return this.ordinal - other.ordinal;
    }

    @SuppressWarnings("unchecked")
    public final Class<E> getDeclaringClass() {
        Class<?> clazz = avm_getClass();
        Class<?> zuper = clazz.avm_getSuperclass();
        return (zuper.getRealClass() == Enum.class) ? (Class<E>)clazz : (Class<E>)zuper;
    }

    public static <T extends Enum<T>> T avm_valueOf(Class<T> enumType,
                                                String name) {
        IInstrumentation.attachedThreadInstrumentation.get().chargeEnergy(RuntimeMethodFeeSchedule.Enum_avm_valueOf);
        return internalValueOf(enumType, name);
    }

    public static <T extends Enum<T>> T internalValueOf(Class<T> enumType, String name) {
        T result = enumType.enumConstantDirectory().get(name);
        if (result != null)
            return result;
        if (name == null)
            throw new NullPointerException("Name is null");
        throw new IllegalArgumentException(
                "No enum constant " + enumType.getName() + "." + name);
    }

    public String getName() {
        lazyLoad();
        return name;
    }

    @SuppressWarnings("deprecation")
    protected final void finalize() { }

}
