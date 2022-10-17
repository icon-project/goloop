package pi;

import foundation.icon.ee.util.IObjects;
import i.IInstrumentation;
import i.IObject;
import org.aion.avm.EnergyCalculator;
import org.aion.avm.RuntimeMethodFeeSchedule;
import s.java.util.Collection;
import s.java.util.Map;
import s.java.util.Set;

// iteration order is deterministic
// may have null value
// confirms standard map hashCode and equals
public class UnmodifiableArrayMap<K extends IObject, V extends IObject>
        extends UnmodifiableArrayContainer
        implements Map<K, V> {
    public UnmodifiableArrayMap(IObject[] keyValue) {
        super(keyValue);
        IInstrumentation.charge(RuntimeMethodFeeSchedule.UnmodifiableArrayMap_constructor);
    }

    public int avm_size() {
        IInstrumentation.charge(RuntimeMethodFeeSchedule.UnmodifiableArrayMap_size);
        return data.length / 2;
    }

    public boolean avm_containsKey(IObject key) {
        IInstrumentation.charge(EnergyCalculator.multiplyLinearValueByMethodFeeLevel1AndAddBase(RuntimeMethodFeeSchedule.UnmodifiableArrayMap_containsKey, data.length / 2));
        return indexOf(key, 0, 2) >= 0;
    }

    public boolean avm_containsValue(IObject value) {
        IInstrumentation.charge(EnergyCalculator.multiplyLinearValueByMethodFeeLevel1AndAddBase(RuntimeMethodFeeSchedule.UnmodifiableArrayMap_containsValue, data.length / 2));
        return indexOf(value, 1, 2) >= 0;
    }

    public V avm_get(IObject key) {
        IInstrumentation.charge(EnergyCalculator.multiplyLinearValueByMethodFeeLevel1AndAddBase(RuntimeMethodFeeSchedule.UnmodifiableArrayMap_get, data.length / 2));
        var index = indexOf(key, 0, 2);
        if (index < 0) {
            return null;
        }
        return (V) data[index + 1];
    }

    public V avm_put(K key, V value) {
        throw new UnsupportedOperationException();
    }

    public V avm_remove(IObject key) {
        throw new UnsupportedOperationException();
    }

    public void avm_putAll(Map<? extends K, ? extends V> m) {
        throw new UnsupportedOperationException();
    }

    private IObject[] collect(int offset) {
        var oa = new IObject[data.length / 2];
        int dst = 0;
        var es = IInstrumentation.getCurrentFrameContext().getExternalState();
        if (!es.fixMapValues()) {
            offset = 0;
        }
        for (int i = offset; i < data.length; i += 2) {
            oa[dst++] = data[i];
        }
        return oa;
    }

    public Set<K> avm_keySet() {
        IInstrumentation.charge(RuntimeMethodFeeSchedule.UnmodifiableArrayMap_keySet);
        return new UnmodifiableArraySet<>(collect(0));
    }

    public Collection<V> avm_values() {
        IInstrumentation.charge(RuntimeMethodFeeSchedule.UnmodifiableArrayMap_values);
        return new UnmodifiableArraySet<>(collect(1));
    }

    public Set<Map.Entry<K, V>> avm_entrySet() {
        IInstrumentation.charge(RuntimeMethodFeeSchedule.UnmodifiableArrayMap_entrySet);
        var oa = new IObject[data.length / 2];
        int dst = 0;
        for (int i = 0; i < data.length; i += 2) {
            oa[dst++] = new UnmodifiableMapEntry<>((K) data[i], (V) data[i + 1]);
        }
        return new UnmodifiableArraySet<>(oa);
    }

    public boolean avm_equals(IObject o) {
        IInstrumentation.charge(EnergyCalculator.multiplyLinearValueByMethodFeeLevel1AndAddBase(RuntimeMethodFeeSchedule.UnmodifiableArrayMap_equals, data.length / 2));
        if (o == this) {
            return true;
        }
        if (!(o instanceof Map)) {
            return false;
        }
        Map<?, ?> m = (Map<?, ?>) o;
        if (m.avm_size() * 2 != data.length) {
            return false;
        }
        try {
            for (int i = 0; i < data.length; i += 2) {
                IObject k = data[i];
                IObject v = data[i + 1];
                if (v == null) {
                    if (!(m.avm_get(k) == null && m.avm_containsKey(k))) {
                        return false;
                    }
                } else {
                    if (!v.avm_equals(m.avm_get(k))) {
                        return false;
                    }
                }
            }
            return true;
        } catch (ClassCastException | NullPointerException ex) {
            return false;
        }
    }

    public int avm_hashCode() {
        IInstrumentation.charge(EnergyCalculator.multiplyLinearValueByMethodFeeLevel1AndAddBase(RuntimeMethodFeeSchedule.UnmodifiableArrayMap_hashCode, data.length / 2));
        int hash = 0;
        for (int i = 0; i < data.length; ) {
            var kh = IObjects.hashCode(data[i++]);
            hash += kh ^ IObjects.hashCode(data[i++]);
        }
        return hash;
    }

    private static final Map<?, ?> EMPTY_MAP =
            new UnmodifiableArrayMap<>(IObjects.EMPTY_ARRAY);

    @SuppressWarnings("unchecked")
    public static <K extends IObject, V extends IObject> Map<K, V> emptyMap() {
        IInstrumentation.charge(RuntimeMethodFeeSchedule.UnmodifiableArrayMap_emptyMap);
        return (Map<K, V>) EMPTY_MAP;
    }

    public UnmodifiableArrayMap(Void ignore, int readIndex) {
        super(ignore, readIndex);
    }
}
