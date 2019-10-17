package avm;

public interface CollectionDB<K, V> extends DictDB<K>, NestingDictDB<K, V>, ArrayDB {
}
