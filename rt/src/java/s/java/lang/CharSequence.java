package s.java.lang;

import i.IObject;

public interface CharSequence extends IObject {

    int avm_length();

    char avm_charAt(int index);

    CharSequence avm_subSequence(int start, int end);

    String avm_toString();
}