package p.avm;

import i.IObject;

public interface NestingDictDB {
    /**
     * Returns Collection for the key. This method shall be called only if
     * type of V is DictDB or ArrayDB.
     *
     * @param key
     * @return
     */
    IObject avm_at(IObject key);
}
