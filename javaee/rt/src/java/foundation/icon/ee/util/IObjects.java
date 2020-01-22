package foundation.icon.ee.util;

import i.IObject;

public class IObjects {
    public static boolean equals(IObject a, IObject b) {
        return a == null ? b == null : a.avm_equals(b);
    }

    public static int hashCode(IObject o) {
        return o != null ? o.avm_hashCode() : 0;
    }

    public static int indexOf(
            IObject[] arr,
            IObject o,
            int start,
            int end,
            int step
    ) {
        if (o == null) {
            for (int i = start; i < end; i += step) {
                if (arr[i] == null) {
                    return i;
                }
            }
        } else {
            for (int i = start; i < end; i += step) {
                if (arr[i].avm_equals(o)) {
                    return i;
                }
            }
        }
        return -1;
    }

    public static int lastIndexOf(IObject[] arr, IObject o) {
        if (o == null) {
            for (int i = arr.length-1; i>=0; i--) {
                if (arr[i] == null) {
                    return i;
                }
            }
        } else {
            for (int i = arr.length-1; i>=0; i--) {
                if (arr[i].avm_equals(o)) {
                    return i;
                }
            }
        }
        return -1;
    }

    public static int hashCode(IObject... a) {
        if (a == null) {
            return 0;
        }
        int result = 1;
        for (IObject e : a) {
            result = 31 * result + IObjects.hashCode(e);
        }
        return result;
    }

    public static final IObject[] EMPTY_ARRAY = new IObject[0];

    public static IObject[] requireNonNullElements(IObject[] elems) {
        for (var e : elems) {
            if (e == null) {
                throw new NullPointerException();
            }
        }
        return elems;
    }
}
