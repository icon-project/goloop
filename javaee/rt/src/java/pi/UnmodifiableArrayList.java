package pi;

import foundation.icon.ee.util.IObjects;
import i.IInstrumentation;
import i.IObject;
import org.aion.avm.EnergyCalculator;
import org.aion.avm.RuntimeMethodFeeSchedule;
import s.java.util.Iterator;
import s.java.util.List;
import s.java.util.ListIterator;

public class UnmodifiableArrayList<E extends IObject>
        extends UnmodifiableArrayCollection<E>
        implements List<E> {
    public UnmodifiableArrayList(IObject[] data) {
        super(data);
        EnergyCalculator.chargeEnergyLevel1(RuntimeMethodFeeSchedule.UnmodifiableArrayList_constructor, data.length);
    }

    public boolean avm_equals(IObject o) {
        IInstrumentation.charge(RuntimeMethodFeeSchedule.UnmodifiableArrayList_equals);
        if (o == this) {
            return true;
        }
        if (!(o instanceof List)) {
            return false;
        }
        Iterator<E> e1 = avm_iterator();
        ListIterator<?> e2 = ((List<?>) o).avm_listIterator();
        while (e1.avm_hasNext() && e2.avm_hasNext()) {
            IObject o1 = e1.avm_next();
            IObject o2 = e2.avm_next();
            if (!IObjects.equals(o1, o2)) {
                return false;
            }
        }
        return !(e1.avm_hasNext() || e2.avm_hasNext());
    }

    public int avm_hashCode() {
        EnergyCalculator.chargeEnergyLevel1(RuntimeMethodFeeSchedule.UnmodifiableArrayList_hashCode, data.length);
        return IObjects.hashCode(data);
    }

    public E avm_get(int index) {
        IInstrumentation.charge(RuntimeMethodFeeSchedule.UnmodifiableArrayList_get);
        return (E) data[index];
    }

    public IObject avm_set(int index, E element) {
        throw new UnsupportedOperationException();
    }

    public void avm_add(int index, E element) {
        throw new UnsupportedOperationException();
    }

    public E avm_remove(int index) {
        throw new UnsupportedOperationException();
    }

    public int avm_indexOf(IObject o) {
        EnergyCalculator.chargeEnergyLevel1(RuntimeMethodFeeSchedule.UnmodifiableArrayList_indexOf, data.length);
        return indexOf(o);
    }

    public int avm_lastIndexOf(IObject o) {
        EnergyCalculator.chargeEnergyLevel1(RuntimeMethodFeeSchedule.UnmodifiableArrayList_lastIndexOf, data.length);
        return lastIndexOf(o);
    }

    public ListIterator<E> avm_listIterator() {
        IInstrumentation.charge(RuntimeMethodFeeSchedule.UnmodifiableArrayList_listIterator);
        return new ListIter();
    }

    public ListIterator<E> avm_listIterator(int index) {
        IInstrumentation.charge(RuntimeMethodFeeSchedule.UnmodifiableArrayList_listIterator);
        return new ListIter(index);
    }

    public List<E> avm_subList(int fromIndex, int toIndex) {
        EnergyCalculator.chargeEnergyLevel1(RuntimeMethodFeeSchedule.UnmodifiableArrayList_subList, data.length);
        return new UnmodifiableArrayList<>(
                java.util.Arrays.copyOfRange(data, fromIndex, toIndex));
    }

    class ListIter extends UnmodifiableArrayCollection<E>.Iter
            implements ListIterator<E> {
        ListIter() {
            super();
        }

        ListIter(int index) {
            super(index);
        }

        public boolean avm_hasPrevious() {
            IInstrumentation.charge(RuntimeMethodFeeSchedule.UnmodifiableArrayList_ListIter_hasPrevious);
            return index > 0;
        }

        public E avm_previous() {
            IInstrumentation.charge(RuntimeMethodFeeSchedule.UnmodifiableArrayList_ListIter_previous);
            return (E) data[--index];
        }

        public int avm_nextIndex() {
            IInstrumentation.charge(RuntimeMethodFeeSchedule.UnmodifiableArrayList_ListIter_nextIndex);
            return index;
        }

        public int avm_previousIndex() {
            IInstrumentation.charge(RuntimeMethodFeeSchedule.UnmodifiableArrayList_ListIter_previousIndex);
            return index - 1;
        }

        public void avm_set(E e) {
            throw new UnsupportedOperationException();
        }

        public void avm_add(E e) {
            throw new UnsupportedOperationException();
        }
    }

    private static final List<?> EMPTY_LIST =
            new UnmodifiableArrayList<>(IObjects.EMPTY_ARRAY);

    @SuppressWarnings("unchecked")
    public static <E extends IObject> List<E> emptyList() {
        IInstrumentation.charge(RuntimeMethodFeeSchedule.UnmodifiableArrayList_emptyList);
        return (List<E>) EMPTY_LIST;
    }

    public UnmodifiableArrayList(Void ignore, int readIndex) {
        super(ignore, readIndex);
    }
}
