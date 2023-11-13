package pi;

import a.ObjectArray;
import i.IInstrumentation;
import i.IObject;
import i.IObjectArray;
import org.aion.avm.EnergyCalculator;
import org.aion.avm.RuntimeMethodFeeSchedule;
import s.java.lang.Object;
import s.java.util.Collection;
import s.java.util.Iterator;

public class UnmodifiableArrayCollection<E extends IObject>
        extends UnmodifiableArrayContainer
        implements Collection<E> {
    UnmodifiableArrayCollection(IObject[] elems) {
        super(elems);
    }

    public int avm_size() {
        IInstrumentation.charge(RuntimeMethodFeeSchedule.UnmodifiableArrayCollection_size);
        return data.length;
    }

    public boolean avm_contains(IObject o) {
        EnergyCalculator.chargeEnergyLevel1(RuntimeMethodFeeSchedule.UnmodifiableArrayCollection_contains, data.length);
        return indexOf(o) >= 0;
    }

    public IObjectArray avm_toArray() {
        EnergyCalculator.chargeEnergyLevel1(RuntimeMethodFeeSchedule.UnmodifiableArrayCollection_toArray, data.length);
        return ObjectArray.newWithCharge(data.clone());
    }

    public boolean avm_add(E e) {
        throw new UnsupportedOperationException();
    }

    public boolean avm_remove(IObject o) {
        throw new UnsupportedOperationException();
    }

    public boolean avm_containsAll(Collection<? extends IObject> c) {
        IInstrumentation.charge(RuntimeMethodFeeSchedule.UnmodifiableArrayCollection_containsAll);
        var iter = c.avm_iterator();
        while (iter.avm_hasNext()) {
            if (!avm_contains(iter.avm_next())) {
                return false;
            }
        }
        return true;
    }

    public boolean avm_addAll(Collection<? extends E> c) {
        throw new UnsupportedOperationException();
    }

    public boolean avm_removeAll(Collection<? extends IObject> c) {
        throw new UnsupportedOperationException();
    }

    public boolean avm_retainAll(Collection<? extends IObject> c) {
        throw new UnsupportedOperationException();
    }

    class Iter extends Object implements Iterator<E> {
        int index;

        Iter() {
        }

        Iter(int index) {
            this.index = index;
        }

        public boolean avm_hasNext() {
            IInstrumentation.charge(RuntimeMethodFeeSchedule.UnmodifiableArrayCollection_Iter_hasNext);
            return index < data.length;
        }

        public E avm_next() {
            IInstrumentation.charge(RuntimeMethodFeeSchedule.UnmodifiableArrayCollection_Iter_next);
            return (E) data[index++];
        }

        public void avm_remove() {
            throw new UnsupportedOperationException();
        }
    }

    public Iterator<E> avm_iterator() {
        IInstrumentation.charge(RuntimeMethodFeeSchedule.UnmodifiableArrayCollection_iterator);
        return new Iter();
    }

    public UnmodifiableArrayCollection(Void ignore, int readIndex) {
        super(ignore, readIndex);
    }
}
