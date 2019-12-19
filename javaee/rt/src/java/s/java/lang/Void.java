package s.java.lang;

import i.ConstantToken;
import i.ShadowClassConstantId;

public final class Void extends Object{

    public static final Class<Void> avm_TYPE = new Class(java.lang.Void.TYPE, new ConstantToken(ShadowClassConstantId.Void_avm_TYPE));

    /*
     * The Void class cannot be instantiated.
     */
    private Void() {
    }
}
