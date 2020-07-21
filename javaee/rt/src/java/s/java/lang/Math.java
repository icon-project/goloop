package s.java.lang;

import i.IInstrumentation;

public final class Math extends Object {
    static {
        // Shadow classes MUST be loaded during bootstrap phase.
        IInstrumentation.attachedThreadInstrumentation.get().bootstrapOnly();
    }

    private Math() {}

    public static int avm_abs(int a) {
        return StrictMath.avm_abs(a);
    }

    public static long avm_abs(long a) {
        return StrictMath.avm_abs(a);
    }

    public static float avm_abs(float a) {
        return StrictMath.avm_abs(a);
    }

    public static double avm_abs(double a) {
        return StrictMath.avm_abs(a);
    }

    public static int avm_max(int a, int b) {
        return StrictMath.avm_max(a, b);
    }

    public static long avm_max(long a, long b) {
        return StrictMath.avm_max(a, b);
    }

    public static float avm_max(float a, float b) {
        return StrictMath.avm_max(a, b);
    }

    public static double avm_max(double a, double b) {
        return StrictMath.avm_max(a, b);
    }

    public static int avm_min(int a, int b) {
        return StrictMath.avm_min(a, b);
    }

    public static long avm_min(long a, long b) {
        return StrictMath.avm_min(a, b);
    }

    public static float avm_min(float a, float b) {
        return StrictMath.avm_min(a, b);
    }

    public static double avm_min(double a, double b) {
        return StrictMath.avm_min(a, b);
    }
}
