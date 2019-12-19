package s.java.lang.invoke;

import i.IInstrumentation;
import i.InvokeDynamicChecks;
import i.RuntimeAssertionError;
import s.java.lang.Integer;
import s.java.lang.Short;
import s.java.lang.String;
import s.java.lang.Long;
import s.java.lang.Double;
import s.java.lang.Float;
import s.java.lang.Character;
import s.java.lang.Byte;
import s.java.lang.Boolean;

import java.lang.invoke.ConstantCallSite;
import java.lang.invoke.MethodHandle;
import java.lang.invoke.MethodHandles;
import java.lang.invoke.MethodType;


public final class StringConcatFactory extends s.java.lang.Object {
    private static final char RECIPE_DYNAMIC_ARGUMENT_FLAG = '\u0001';
    private static final char RECIPE_STATIC_ARGUMENT_FLAG = '\u0002';

    static {
        // Shadow classes MUST be loaded during bootstrap phase.
        IInstrumentation.attachedThreadInstrumentation.get().bootstrapOnly();
    }

    public static String avm_concat(java.lang.String recipe, Object[] staticArgs, Object[] dynamicArgs) {
        // Note that we want to use a shadow StringBuilder since it correctly calls avm_toString() as opposed to toString().
        // (note that this will allocate a new object, at the level of the DApp, but only in the same way the non-invokedynamic approach would).
        final s.java.lang.StringBuilder builder = new s.java.lang.StringBuilder();
        int staticArgsIdx = 0;
        int dynamicArgsIdx = 0;
        for (int idx = 0; idx < recipe.length(); idx++) {
            char ch = recipe.charAt(idx);
            if (ch == RECIPE_DYNAMIC_ARGUMENT_FLAG) {
                s.java.lang.Object arg = mapBoxedType(dynamicArgs[dynamicArgsIdx++], false);
                builder.avm_append(arg);
            } else if (ch == RECIPE_STATIC_ARGUMENT_FLAG) {
                s.java.lang.Object arg = mapBoxedType(staticArgs[staticArgsIdx++], true);
                builder.avm_append(arg);
            } else {
                builder.avm_append(ch);
            }
        }
        return builder.avm_toString();
    }

    private static s.java.lang.Object mapBoxedType(Object obj, boolean isStaticArg){
        s.java.lang.Object ret = null;
        if (null == obj){
            ret = null;
        }else if (obj instanceof s.java.lang.Object){
            ret = (s.java.lang.Object)obj;
        }else {
            Class argClass = obj.getClass();
            if(argClass.equals(java.lang.Short.class)) {
                ret = Short.avm_valueOf(((java.lang.Short) obj));
            } else if(argClass.equals(java.lang.Integer.class)) {
                ret = Integer.avm_valueOf(((java.lang.Integer) obj));
            }else if(argClass.equals(java.lang.Long.class)) {
                ret = Long.avm_valueOf(((java.lang.Long) obj));
            }else if(argClass.equals(java.lang.Float.class)) {
                ret = Float.avm_valueOf(((java.lang.Float) obj));
            }else if(argClass.equals(java.lang.Double.class)) {
                ret = Double.avm_valueOf(((java.lang.Double) obj));
            }else if(argClass.equals(java.lang.Boolean.class)) {
                ret = Boolean.avm_valueOf(((java.lang.Boolean) obj));
            }else if(argClass.equals(java.lang.Byte.class)) {
                ret = Byte.avm_valueOf(((java.lang.Byte) obj));
            }else if(argClass.equals(java.lang.Character.class)) {
                ret = Character.avm_valueOf(((java.lang.Character) obj));
            }else if(isStaticArg && argClass.equals(java.lang.String.class)) {
                ret = String.avm_valueOf(new String((java.lang.String)obj));
            }else {
                RuntimeAssertionError.unreachable("String concat receives unexpected type " + argClass.getName());
            }
        }
        return ret;
    }

    /**
     * A bootstrap method for handling string concatenation
     *
     * @see java.lang.invoke.StringConcatFactory#makeConcatWithConstants(MethodHandles.Lookup, java.lang.String, MethodType, java.lang.String, Object...)
     */
    public static java.lang.invoke.CallSite avm_makeConcatWithConstants(
            java.lang.invoke.MethodHandles.Lookup owner,
            java.lang.String invokedName,
            MethodType concatType,
            java.lang.String recipe,
            Object... constants) throws NoSuchMethodException, IllegalAccessException {
        InvokeDynamicChecks.checkOwner(owner);
        // Note that we currently only use the avm_makeConcatWithConstants invoked name.
        RuntimeAssertionError.assertTrue("avm_makeConcatWithConstants".equals(invokedName));
        
        final MethodType concatMethodType = MethodType.methodType(
                String.class, // NOTE! First arg is return value
                java.lang.String.class,
                Object[].class,
                Object[].class);
        final MethodHandle concatMethodHandle = owner
                .findStatic(StringConcatFactory.class, "avm_concat", concatMethodType)
                .bindTo(recipe)
                .bindTo(constants)
                .asVarargsCollector(Object[].class)
                .asType(concatType);
        return new ConstantCallSite(concatMethodHandle);
    }

    // Cannot be instantiated.
    private StringConcatFactory() {
    }
    // Note:  No instances can be created so no deserialization constructor required.
}
