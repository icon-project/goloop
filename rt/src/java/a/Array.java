package a;

import i.IInstrumentation;
import i.IObject;
import s.java.lang.Cloneable;
import s.java.lang.Object;


public abstract class Array extends Object implements Cloneable, IArray {
    // Initial creation.
    public Array() {
    }

    // Deserializer support.
    public Array(Void ignore, int readIndex) {
        super(ignore, readIndex);
    }

    public abstract java.lang.Object getUnderlyingAsObject();

    public abstract void setUnderlyingAsObject(java.lang.Object u);

    public abstract java.lang.Object getAsObject(int idx);

    public abstract int length();

    public abstract IObject avm_clone();

    /**
     * Note that this helper exists primarily so it can be called by generated/instrumented code.
     * Since the array code is not generally in the same class loader of the DApp, it can't call the runtime class, directly.
     * 
     * @param cost The energy cost to charge the current DApp.
     */
    static protected void chargeEnergy(long cost){
        IInstrumentation.attachedThreadInstrumentation.get().chargeEnergy(cost);
    }
}
