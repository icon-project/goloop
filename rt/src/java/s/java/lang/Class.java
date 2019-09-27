package s.java.lang;

import org.aion.avm.ClassNameExtractor;
import a.ObjectArray;
import i.AvmThrowable;
import i.ConstantToken;
import i.IInstrumentation;
import i.IObject;

import java.lang.reflect.InvocationTargetException;
import java.lang.reflect.Method;
import java.util.HashMap;
import java.util.Map;

import org.aion.avm.RuntimeMethodFeeSchedule;
import s.java.io.Serializable;

public final class Class<T> extends Object implements Serializable {
    static {
        // Shadow classes MUST be loaded during bootstrap phase.
        IInstrumentation.attachedThreadInstrumentation.get().bootstrapOnly();
    }

    public String avm_getName() {
        IInstrumentation.attachedThreadInstrumentation.get().chargeEnergy(RuntimeMethodFeeSchedule.Class_avm_getName);
        // Note that we actively try not to give the same instance of the name wrapper back (since the user could see implementation details of our
        // contract life-cycle or the underlying JVM/ClassLoader.
        return getName();
    }

    public String avm_toString() {
        IInstrumentation.attachedThreadInstrumentation.get().chargeEnergy(RuntimeMethodFeeSchedule.Class_avm_toString);
        return new String((this.v.isInterface() ? "interface " : (this.v.isPrimitive() ? "" : "class "))
                + getName());
    }

    public IObject avm_cast(IObject obj) {
        IInstrumentation.attachedThreadInstrumentation.get().chargeEnergy(RuntimeMethodFeeSchedule.Class_avm_cast);
        return (IObject)this.v.cast(obj);
    }

    public java.lang.Class<T> getRealClass(){return this.v;}

    @SuppressWarnings("unchecked")
    public Class<T> avm_getSuperclass() {
        IInstrumentation.attachedThreadInstrumentation.get().chargeEnergy(RuntimeMethodFeeSchedule.Class_avm_getSuperclass);
        // Note that we need to return null if the underlying is the shadow object root.
        Class<T> toReturn = null;
        if (s.java.lang.Object.class != this.v) {
            toReturn = (Class<T>) IInstrumentation.attachedThreadInstrumentation.get().wrapAsClass(this.v.getSuperclass());
        }
        return toReturn;
    }

    public boolean avm_desiredAssertionStatus() {
        IInstrumentation.attachedThreadInstrumentation.get().chargeEnergy(RuntimeMethodFeeSchedule.Class_avm_desiredAssertionStatus);
        // Note that we currently handle assertions as always-enabled.
        // Internally, these will result in throwing AssertionError which, unless caught by the user's code, results in a FAILED_EXCEPTION status.
        // See issue-72 for more details on our thought process and future interpretations of this we may want to entertain.
        return true;
    }

    //=======================================================
    // Methods below are used by Enum
    //========================================================
    Map<String, T> enumConstantDirectory() {
        Map<String, T> directory = enumConstantDirectory;
        if (directory == null) {
            ObjectArray universe = getEnumConstantsShared();
            if (universe == null)
                throw new IllegalArgumentException(
                        avm_getName() + " is not an enum type");
            directory = new HashMap<>(2 * universe.length());
            for (int i = 0; i < universe.length(); i++){
                @SuppressWarnings("unchecked")
                T constant = (T) universe.get(i);
                directory.put(((Enum<?>)constant).avm_name(), constant);
            }
            enumConstantDirectory = directory;
        }
        return directory;
    }
    private transient volatile Map<String, T> enumConstantDirectory;



    ObjectArray getEnumConstantsShared() {
        ObjectArray constants = enumConstants;
        if (constants == null) {
            try {
                Method m = v.getDeclaredMethod("avm_values");
                java.lang.Object value = m.invoke(null);

                constants = (ObjectArray) value;
                enumConstants = constants;
            } catch (InvocationTargetException e) {
                // This can happen as a result of an out-of-energy exception - throw back any of our types.
                if (e.getCause() instanceof AvmThrowable) {
                    throw (AvmThrowable) e.getCause();
                } else {
                    // This is unexpected, but the user should be able to craft an attempt to call this so just log it, in case this is a cause for concern.
                    e.printStackTrace();
                    constants = null;
                }
            } catch (NoSuchMethodException | IllegalAccessException e) {
                // This is unexpected, but the user should be able to craft an attempt to call this so just log it, in case this is a cause for concern.
                e.printStackTrace();
                constants = null;
            }
        }
        return constants;
    }
    private transient volatile ObjectArray enumConstants;


    //=======================================================
    // Methods below are used by runtime and test code only!
    //========================================================

    public Class(java.lang.Class<T> v) {
        // We will base our hashcode on the original class name.
        super(null, null, ClassNameExtractor.getOriginalClassName(v.getName()).hashCode());
        this.v = v;
    }

    protected Class(java.lang.Class<T> v, ConstantToken constantToken) {
        super(constantToken);
        this.v = v;
    }
    private final java.lang.Class<T> v;

    @Override
    public java.lang.String toString() {
        return this.v.toString();
    }

    public String getName() {
        return new s.java.lang.String(ClassNameExtractor.getOriginalClassName(v.getName()));
    }
}
