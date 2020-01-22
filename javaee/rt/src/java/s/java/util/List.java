package s.java.util;

import a.ObjectArray;
import pi.UnmodifiableArrayList;
import foundation.icon.ee.util.IObjects;
import i.IObject;
import i.IObjectArray;

import java.util.Objects;

public interface List<E extends IObject> extends Collection<E> {

    // Positional Access Operations

    E avm_get(int index);

    IObject avm_set(int index, E element);

    void avm_add(int index, E element);

    E avm_remove(int index);

    int avm_indexOf(IObject o);

    int avm_lastIndexOf(IObject o);

    ListIterator<E> avm_listIterator();

    ListIterator<E> avm_listIterator(int index);

    // View

    List<E> avm_subList(int fromIndex, int toIndex);

    static <E extends IObject> List<E> avm_of() {
        return UnmodifiableArrayList.emptyList();
    }

    static <E extends IObject> List<E> avm_of(E o0) {
        return new UnmodifiableArrayList<>(IObjects.requireNonNullElements(
                new IObject[]{o0}));
    }

    static <E extends IObject> List<E> avm_of(E o0, E o1) {
        return new UnmodifiableArrayList<>(IObjects.requireNonNullElements(
                new IObject[]{o0, o1}));
    }

    static <E extends IObject> List<E> avm_of(E o0, E o1, E o2) {
        return new UnmodifiableArrayList<>(IObjects.requireNonNullElements(
                new IObject[]{o0, o1, o2}));
    }

    static <E extends IObject> List<E> avm_of(E o0, E o1, E o2, E o3) {
        return new UnmodifiableArrayList<>(IObjects.requireNonNullElements(
                new IObject[]{o0, o1, o2, o3}));
    }

    static <E extends IObject> List<E> avm_of(E o0, E o1, E o2, E o3,
                                              E o4) {
        return new UnmodifiableArrayList<>(IObjects.requireNonNullElements(
                new IObject[]{o0, o1, o2, o3, o4}));
    }

    static <E extends IObject> List<E> avm_of(E o0, E o1, E o2, E o3,
                                              E o4, E o5) {
        return new UnmodifiableArrayList<>(IObjects.requireNonNullElements(
                new IObject[]{o0, o1, o2, o3, o4, o5}));
    }

    static <E extends IObject> List<E> avm_of(E o0, E o1, E o2, E o3,
                                              E o4, E o5, E o6) {
        return new UnmodifiableArrayList<>(IObjects.requireNonNullElements(
                new IObject[]{o0, o1, o2, o3, o4, o5, o6}));
    }

    static <E extends IObject> List<E> avm_of(E o0, E o1, E o2, E o3,
                                              E o4, E o5, E o6, E o7) {
        return new UnmodifiableArrayList<>(IObjects.requireNonNullElements(
                new IObject[]{o0, o1, o2, o3, o4, o5, o6, o7}));
    }

    static <E extends IObject> List<E> avm_of(E o0, E o1, E o2, E o3,
                                              E o4, E o5, E o6, E o7,
                                              E o8) {
        return new UnmodifiableArrayList<>(IObjects.requireNonNullElements(
                new IObject[]{o0, o1, o2, o3, o4, o5, o6, o7, o8}));
    }

    static <E extends IObject> List<E> avm_of(E o0, E o1, E o2, E o3,
                                              E o4, E o5, E o6, E o7,
                                              E o8, E o9) {
        return new UnmodifiableArrayList<>(IObjects.requireNonNullElements(
                new IObject[]{o0, o1, o2, o3, o4, o5, o6, o7, o8, o9}));
    }

    static <E extends IObject> List<E> avm_of(IObjectArray elements) {
        var oa = ((ObjectArray) elements).getUnderlying();
        var data = new IObject[oa.length];
        for (int i = 0; i < oa.length; i++) {
            data[i] = Objects.requireNonNull((IObject) oa[i]);
        }
        return new UnmodifiableArrayList<>(data);
    }
}
