package i;


import a.IArray;

/**
 * This interface contains the common ObjectArray methods callable from transformed code.
 * Originally, these calls were invokevirtual against the ObjectArray instance, directly, but some type unification rules around arrays and interfaces
 * required that we often use an interface and call against it using invokeinterface.  This interface declares those requirements.
 * It also needs to extend IObject since even unifying array contexts need to ultimately unify to object.
 * See issue-82 for more details.
 */
public interface IObjectArray extends IObject, IArray {
    public Object get(int idx);

    public void set(int idx, Object val);

    public IObject avm_clone();
}
