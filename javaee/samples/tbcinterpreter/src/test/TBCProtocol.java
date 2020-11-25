package test;

/**
 *  Test byte code protocol
 */
public class TBCProtocol {
    //
    //  Common Instructions
    //

    //  address : to
    //  int16 : code length
    //  byte[] : code
    public static final byte CALL = 0;

    //  int16 : revert code. unused
    public static final byte REVERT = 1;

    //
    //  Java Specific Instructions
    //

    public static final byte MAX_VAR = 8;

    public static final byte VAR_TYPE_STATIC = 0;
    public static final byte VAR_TYPE_INSTANCE = 1;
    public static final byte VAR_TYPE_LOCAL = 2;

    //  int8 var type
    //  int8 var id
    //  int16 value len. -1 if value is null
    //  byte[] value
    public static final byte SET = 2;

    //  int8 var type
    //  int8 var id
    //  int16 value len
    //  byte[] value
    public static final byte APPEND = 3;

    //  int8 var type
    //  int8 var id
    //  int16 value len
    //  byte[] value
    //  int8 comparison operator
    public static final byte EXPECT = 4;

    public static final byte CMP_EQ = 0;
    public static final byte CMP_NE = 1;

    //  int8 var type
    //  int8 var id
    //  int8 var type
    //  int8 var id
    public static final byte SET_REF = 5;

    //  int8 var type
    //  int8 var id
    //  int8 var type
    //  int8 var id
    //  int8 comparison operator
    public static final byte EXPECT_REF = 6;
}
